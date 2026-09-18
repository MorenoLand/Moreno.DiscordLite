//go:build windows

package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"golang.org/x/sys/windows"
)

const maxRPCPayload = 16 * 1024 * 1024

func startDiscordRPCBridge(app *discordApp) {
	for i := uint8(0); i < 10; i++ {
		go listenRPCPipe(app, i)
	}
}

func listenRPCPipe(app *discordApp, index uint8) {
	for {
		pipe, err := createRPCPipe(index)
		if err != nil {
			time.Sleep(1 * time.Second)
			continue
		}
		connectErr := windows.ConnectNamedPipe(pipe, nil)
		if connectErr == nil || errors.Is(connectErr, windows.ERROR_PIPE_CONNECTED) {
			go func(p windows.Handle) {
				defer windows.DisconnectNamedPipe(p)
				defer windows.CloseHandle(p)
				_ = serveRPCClient(app, p)
			}(pipe)
		} else {
			_ = windows.CloseHandle(pipe)
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func createRPCPipe(index uint8) (windows.Handle, error) {
	name, err := windows.UTF16PtrFromString(fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, index))
	if err != nil {
		return windows.InvalidHandle, err
	}
	pipe, err := windows.CreateNamedPipe(name, windows.PIPE_ACCESS_DUPLEX, windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT, windows.PIPE_UNLIMITED_INSTANCES, 65536, 65536, 0, nil)
	if err != nil {
		return windows.InvalidHandle, err
	}
	return pipe, nil
}

func serveRPCClient(app *discordApp, pipe windows.Handle) error {
	var clientID string
	var currentSocketID string
	hasActivity := false

	defer func() {
		if hasActivity && currentSocketID != "" && app != nil {
			app.setRPCActivity(currentSocketID, 0, nil)
		}
	}()

	for {
		op, payload, err := readRPCFrame(pipe)
		if err != nil {
			return err
		}
		var value map[string]any
		if json.Unmarshal(payload, &value) != nil {
			value = map[string]any{}
		}
		switch op {
		case 0:
			clientID, _ = value["client_id"].(string)
			if clientID == "" {
				clientID = "0"
			}
			currentSocketID = "RPC:" + clientID
			if err := writeRPCFrame(pipe, 1, map[string]any{
				"cmd": "DISPATCH",
				"evt": "READY",
				"data": map[string]any{
					"v": 1,
					"config": map[string]any{
						"cdn_host":     "cdn.discordapp.com",
						"api_endpoint": "//discord.com/api",
						"environment":  "production",
					},
					"user": map[string]any{
						"id":            "0",
						"username":      "Discord",
						"discriminator": "0000",
						"avatar":        nil,
						"bot":           false,
					},
					"client_id": clientID,
				},
			}); err != nil {
				return err
			}
		case 1:
			if err := handleRPCCommand(app, currentSocketID, clientID, pipe, value, &hasActivity); err != nil {
				return err
			}
		case 3:
			if err := writeRPCFrame(pipe, 4, value); err != nil {
				return err
			}
		}
	}
}

func handleRPCCommand(app *discordApp, socketID, clientID string, pipe windows.Handle, value map[string]any, hasActivity *bool) error {
	command, _ := value["cmd"].(string)
	nonce := value["nonce"]
	response := map[string]any{"cmd": command, "data": nil, "evt": nil, "nonce": nonce}
	switch command {
	case "SET_ACTIVITY":
		args, _ := value["args"].(map[string]any)
		act := args["activity"]
		pid := args["pid"]
		if act == nil {
			if *hasActivity && socketID != "" && app != nil {
				app.setRPCActivity(socketID, pid, nil)
			}
			*hasActivity = false
			response["data"] = nil
		} else {
			if actMap, ok := act.(map[string]any); ok {
				if _, hasAppID := actMap["application_id"]; !hasAppID && clientID != "" {
					actMap["application_id"] = clientID
				}
			}
			if socketID != "" && app != nil {
				app.setRPCActivity(socketID, pid, act)
				*hasActivity = true
			}
			response["data"] = act
		}
	case "SUBSCRIBE", "UNSUBSCRIBE":
	case "AUTHORIZE":
		response["data"] = map[string]any{"code": "discord-app-local"}
	case "AUTHENTICATE":
		response["evt"] = "ERROR"
		response["data"] = map[string]any{"code": 4000, "message": "authentication unavailable"}
	default:
		response["evt"] = "ERROR"
		response["data"] = map[string]any{"code": 4000, "message": "unknown command"}
	}
	return writeRPCFrame(pipe, 1, response)
}

func readRPCFrame(pipe windows.Handle) (uint32, []byte, error) {
	header := make([]byte, 8)
	if err := readExactPipe(pipe, header); err != nil {
		return 0, nil, err
	}
	op := binary.LittleEndian.Uint32(header[:4])
	length := binary.LittleEndian.Uint32(header[4:])
	if length > maxRPCPayload {
		return 0, nil, fmt.Errorf("rpc frame too large")
	}
	payload := make([]byte, length)
	if err := readExactPipe(pipe, payload); err != nil {
		return 0, nil, err
	}
	return op, payload, nil
}

func readExactPipe(pipe windows.Handle, buffer []byte) error {
	for offset := 0; offset < len(buffer); {
		var read uint32
		if err := windows.ReadFile(pipe, buffer[offset:], &read, nil); err != nil {
			return err
		}
		if read == 0 {
			return io.EOF
		}
		offset += int(read)
	}
	return nil
}

func writeRPCFrame(pipe windows.Handle, op uint32, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(payload) > maxRPCPayload {
		return fmt.Errorf("rpc response too large")
	}
	frame := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(frame[:4], op)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(payload)))
	copy(frame[8:], payload)
	for offset := 0; offset < len(frame); {
		var written uint32
		if err := windows.WriteFile(pipe, frame[offset:], &written, nil); err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		offset += int(written)
	}
	return nil
}
