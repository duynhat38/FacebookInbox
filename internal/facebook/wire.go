package facebook

import (
	"encoding/binary"
)

// wire.go — binary protocol builders for MQTT-over-WebSocket (edge-chat) and
// the DGW LightSpeed binary frame (gateway.facebook.com/ws/lightspeed).

// buildLightspeedWSFrame builds the 2-packet DGW binary frame.
// Mirrors Python _build_lightspeed_ws_frame() exactly (verified byte-for-byte
// against packet captures):
//
//	pkt_a: companion frame — [0x0F][seq][0x00][0x02][0x00][0x00][{][}]  (8 bytes)
//	pkt_b: data frame     — [0x0D][seq][0x00][length LE 2B][0x00][0x00][0x80][json]
//	       where length = 2 + len(json)
func buildLightspeedWSFrame(seq byte, taskJSON []byte) []byte {
	pktA := []byte{0x0F, seq, 0x00, 0x02, 0x00, 0x00, 0x7B, 0x7D}

	length := uint16(2 + len(taskJSON))
	pktB := make([]byte, 0, 3+2+3+len(taskJSON))
	pktB = append(pktB, 0x0D, seq, 0x00)
	pktB = binary.LittleEndian.AppendUint16(pktB, length)
	pktB = append(pktB, 0x00, 0x00, 0x80)
	pktB = append(pktB, taskJSON...)

	return append(pktA, pktB...)
}

// buildMQTTConnect builds an MQTT 3.1 CONNECT packet.
// Mirrors Python _build_mqtt_connect(): protocol "MQIsdp", level 3,
// connect flags 0x82 (username + clean session), keep-alive 15s.
// All real auth is in the Cookie header; username is only session metadata.
func buildMQTTConnect(usernameJSON []byte) []byte {
	const keepAlive = 15
	clientID := []byte("mqttwsclient") // _MQTT_CLIENT_ID
	protocol := []byte("MQIsdp")

	var body []byte
	body = appendU16BE(body, uint16(len(protocol)))
	body = append(body, protocol...)
	body = append(body, 0x03) // protocol level 3 = MQTT 3.1
	body = append(body, 0x82) // connect flags: username set + clean session
	body = appendU16BE(body, keepAlive)
	body = appendU16BE(body, uint16(len(clientID)))
	body = append(body, clientID...)
	body = appendU16BE(body, uint16(len(usernameJSON)))
	body = append(body, usernameJSON...)

	result := []byte{0x10}
	result = append(result, mqttVarLen(len(body))...)
	return append(result, body...)
}

// buildMQTTSubscribe builds an MQTT 3.1 SUBSCRIBE packet for one topic.
// Mirrors Python _build_mqtt_subscribe().
func buildMQTTSubscribe(topic []byte, packetID uint16) []byte {
	var body []byte
	body = appendU16BE(body, packetID)
	body = appendU16BE(body, uint16(len(topic)))
	body = append(body, topic...)
	body = append(body, 0x00) // QoS 0

	result := []byte{0x82}
	result = append(result, mqttVarLen(len(body))...)
	return append(result, body...)
}

// mqttVarLen encodes n as MQTT variable-length (1–4 bytes).
// Mirrors Python _mqtt_remaining_length().
func mqttVarLen(n int) []byte {
	var out []byte
	for {
		b := n % 128
		n /= 128
		if n > 0 {
			b |= 0x80
		}
		out = append(out, byte(b))
		if n == 0 {
			break
		}
	}
	return out
}

func appendU16BE(b []byte, v uint16) []byte {
	return append(b, byte(v>>8), byte(v))
}
