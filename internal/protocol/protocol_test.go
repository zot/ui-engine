// Test Design: test-VariableProtocol.md, test-Communication.md
// CRC: crc-MessageBatcher.md, crc-ProtocolHandler.md
// Spec: protocol.md
package protocol

import (
	"encoding/json"
	"testing"
)

// TestParsePrioritySuffix verifies :high/:med/:low suffix parsing
func TestParsePrioritySuffix(t *testing.T) {
	tests := []struct {
		input    string
		wantName string
		wantPri  Priority
	}{
		{"viewdefs:high", "viewdefs", PriorityHigh},
		{"data:med", "data", PriorityMedium},
		{"optional:low", "optional", PriorityLow},
		{"name", "name", PriorityMedium}, // default
		{"type", "type", PriorityMedium}, // no suffix = medium
	}

	for _, tt := range tests {
		name, pri := ParsePrioritySuffix(tt.input)
		if name != tt.wantName {
			t.Errorf("ParsePrioritySuffix(%q) name = %q, want %q", tt.input, name, tt.wantName)
		}
		if pri != tt.wantPri {
			t.Errorf("ParsePrioritySuffix(%q) priority = %d, want %d", tt.input, pri, tt.wantPri)
		}
	}
}

// TestMessageBatcherQueueValue verifies value queuing with priority
func TestMessageBatcherQueueValue(t *testing.T) {
	b := NewMessageBatcher()

	if !b.IsEmpty() {
		t.Error("New batcher should be empty")
	}

	b.QueueValue(1, json.RawMessage(`"hello"`), PriorityHigh)

	if b.IsEmpty() {
		t.Error("Batcher should not be empty after QueueValue")
	}
}

// TestMessageBatcherQueueProperty verifies property queuing
func TestMessageBatcherQueueProperty(t *testing.T) {
	b := NewMessageBatcher()

	b.QueueProperty(5, "viewdefs:high", `{"Person.DEFAULT":"<template>..."}`)
	b.QueueProperty(5, "type", "Person")

	messages := b.Flush()
	if len(messages) != 2 { // one high, one medium
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

// TestMessageBatcherBuildsPriorityOrderedBatch verifies batch ordering
func TestMessageBatcherBuildsPriorityOrderedBatch(t *testing.T) {
	b := NewMessageBatcher()

	// Queue in reverse priority order
	b.QueueValue(10, json.RawMessage(`"low"`), PriorityLow)
	b.QueueValue(5, json.RawMessage(`"medium"`), PriorityMedium)
	b.QueueValue(1, json.RawMessage(`"high"`), PriorityHigh)

	messages := b.Flush()
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}

	// First should be high priority (var 1)
	var update1 UpdateMessage
	json.Unmarshal(messages[0].Data, &update1)
	if update1.VarID != 1 {
		t.Errorf("First message should be var 1 (high), got var %d", update1.VarID)
	}

	// Second should be medium priority (var 5)
	var update2 UpdateMessage
	json.Unmarshal(messages[1].Data, &update2)
	if update2.VarID != 5 {
		t.Errorf("Second message should be var 5 (medium), got var %d", update2.VarID)
	}

	// Third should be low priority (var 10)
	var update3 UpdateMessage
	json.Unmarshal(messages[2].Data, &update3)
	if update3.VarID != 10 {
		t.Errorf("Third message should be var 10 (low), got var %d", update3.VarID)
	}
}

// TestMessageBatcherSameVariableDifferentPriorities verifies multi-priority for single variable
func TestMessageBatcherSameVariableDifferentPriorities(t *testing.T) {
	b := NewMessageBatcher()

	// Variable 5: high priority property, medium priority value
	b.QueueProperty(5, "viewdefs:high", `{"Test":"..."}`)
	b.QueueValue(5, json.RawMessage(`{"name":"John"}`), PriorityMedium)

	messages := b.Flush()
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages for var 5, got %d", len(messages))
	}

	// First should be high priority (property only)
	var first UpdateMessage
	json.Unmarshal(messages[0].Data, &first)
	if first.VarID != 5 {
		t.Errorf("Expected var 5, got %d", first.VarID)
	}
	if first.Properties == nil || first.Properties["viewdefs"] == "" {
		t.Error("First message should have viewdefs property")
	}
	if first.Value != nil {
		t.Error("First message should not have value")
	}

	// Second should be medium priority (value)
	var second UpdateMessage
	json.Unmarshal(messages[1].Data, &second)
	if second.VarID != 5 {
		t.Errorf("Expected var 5, got %d", second.VarID)
	}
	if second.Value == nil {
		t.Error("Second message should have value")
	}
}

// TestMessageBatcherFlushClearsState verifies flush empties queue
func TestMessageBatcherFlushClearsState(t *testing.T) {
	b := NewMessageBatcher()

	b.QueueValue(1, json.RawMessage(`"test"`), PriorityMedium)
	b.QueueProperty(2, "type", "Person")

	if b.IsEmpty() {
		t.Error("Should not be empty before flush")
	}

	messages := b.Flush()
	if len(messages) == 0 {
		t.Error("Flush should return messages")
	}

	if !b.IsEmpty() {
		t.Error("Should be empty after flush")
	}

	// Second flush returns nil
	messages = b.Flush()
	if messages != nil {
		t.Error("Second flush should return nil")
	}
}

// TestMessageBatcherQueueProperties verifies bulk property queuing
func TestMessageBatcherQueueProperties(t *testing.T) {
	b := NewMessageBatcher()

	props := map[string]string{
		"type":          "Contact",
		"viewdefs:high": `{"Contact.DEFAULT":"..."}`,
	}
	b.QueueProperties(1, props)

	messages := b.Flush()
	// Should be 2 messages: high (viewdefs) and medium (type)
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}
}

// TestMessageBatcherFlushJSON verifies JSON output
func TestMessageBatcherFlushJSON(t *testing.T) {
	b := NewMessageBatcher()

	b.QueueValue(1, json.RawMessage(`"test"`), PriorityMedium)

	jsonBytes, err := b.FlushJSON()
	if err != nil {
		t.Fatalf("FlushJSON error: %v", err)
	}
	if jsonBytes == nil {
		t.Fatal("Expected JSON output")
	}

	// Should be a single message (not array)
	var msg Message
	if err := json.Unmarshal(jsonBytes, &msg); err != nil {
		t.Errorf("Single message should not be wrapped in array: %v", err)
	}
	if msg.Type != MsgUpdate {
		t.Errorf("Expected update message, got %s", msg.Type)
	}
}

// TestMessageBatcherFlushJSONMultiple verifies JSON array for multiple messages
func TestMessageBatcherFlushJSONMultiple(t *testing.T) {
	b := NewMessageBatcher()

	b.QueueValue(1, json.RawMessage(`"first"`), PriorityHigh)
	b.QueueValue(2, json.RawMessage(`"second"`), PriorityMedium)

	jsonBytes, err := b.FlushJSON()
	if err != nil {
		t.Fatalf("FlushJSON error: %v", err)
	}

	// Should be an array
	var messages []Message
	if err := json.Unmarshal(jsonBytes, &messages); err != nil {
		t.Fatalf("Multiple messages should be JSON array: %v", err)
	}
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages in array, got %d", len(messages))
	}
}

// TestNewMessage verifies message creation
func TestNewMessage(t *testing.T) {
	update := UpdateMessage{
		VarID: 42,
		Value: json.RawMessage(`"hello"`),
	}

	msg, err := NewMessage(MsgUpdate, update)
	if err != nil {
		t.Fatalf("NewMessage error: %v", err)
	}

	if msg.Type != MsgUpdate {
		t.Errorf("Expected type %s, got %s", MsgUpdate, msg.Type)
	}

	var decoded UpdateMessage
	if err := json.Unmarshal(msg.Data, &decoded); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}
	if decoded.VarID != 42 {
		t.Errorf("Expected varID 42, got %d", decoded.VarID)
	}
}

// TestMessageEncode verifies message encoding
func TestMessageEncode(t *testing.T) {
	update := UpdateMessage{
		VarID: 1,
		Value: json.RawMessage(`{"name":"John"}`),
		Properties: map[string]string{
			"type": "Person",
		},
	}

	msg, _ := NewMessage(MsgUpdate, update)
	encoded, err := msg.Encode()
	if err != nil {
		t.Fatalf("Encode error: %v", err)
	}

	// Decode back
	var decoded Message
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Type != MsgUpdate {
		t.Errorf("Expected type %s, got %s", MsgUpdate, decoded.Type)
	}
}

// TestParseMessage verifies message parsing
func TestParseMessage(t *testing.T) {
	// Create message
	createData := CreateMessage{
		ParentID: 0,
		Value:    json.RawMessage(`"hello"`),
		Properties: map[string]string{
			"type": "String",
		},
	}
	msg, _ := NewMessage(MsgCreate, createData)
	encoded, _ := msg.Encode()

	// Parse it back
	parsed, err := ParseMessage(encoded)
	if err != nil {
		t.Fatalf("ParseMessage error: %v", err)
	}
	if parsed.Type != MsgCreate {
		t.Errorf("Expected type %s, got %s", MsgCreate, parsed.Type)
	}

	var decoded CreateMessage
	if err := json.Unmarshal(parsed.Data, &decoded); err != nil {
		t.Fatalf("Failed to decode data: %v", err)
	}
	if decoded.ParentID != 0 {
		t.Errorf("Expected parentID 0, got %d", decoded.ParentID)
	}
}

// TestMessageTypes verifies all message types can be created
func TestMessageTypes(t *testing.T) {
	tests := []struct {
		msgType MessageType
		data    interface{}
	}{
		{MsgCreate, CreateMessage{ParentID: 0, Value: json.RawMessage(`null`)}},
		{MsgDestroy, DestroyMessage{VarID: 1}},
		{MsgUpdate, UpdateMessage{VarID: 1, Value: json.RawMessage(`"test"`)}},
		{MsgWatch, WatchMessage{VarID: 1}},
		{MsgUnwatch, WatchMessage{VarID: 1}},
		{MsgGet, GetMessage{VarIDs: []int64{1, 2, 3}}},
		{MsgGetObjects, GetObjectsMessage{ObjIDs: []int64{1, 2}}},
		{MsgPoll, PollMessage{Wait: "5s"}},
		{MsgError, ErrorMessage{VarID: 1, Code: "TEST", Description: "test error"}},
	}

	for _, tt := range tests {
		msg, err := NewMessage(tt.msgType, tt.data)
		if err != nil {
			t.Errorf("NewMessage(%s) error: %v", tt.msgType, err)
			continue
		}
		if msg.Type != tt.msgType {
			t.Errorf("Expected type %s, got %s", tt.msgType, msg.Type)
		}
	}
}

// TestUpdateMessageWithProperties verifies property-only updates
func TestUpdateMessageWithProperties(t *testing.T) {
	update := UpdateMessage{
		VarID: 5,
		Properties: map[string]string{
			"type":     "Contact",
			"inactive": "true",
		},
	}

	msg, _ := NewMessage(MsgUpdate, update)
	encoded, _ := msg.Encode()

	parsed, _ := ParseMessage(encoded)
	var decoded UpdateMessage
	json.Unmarshal(parsed.Data, &decoded)

	if decoded.VarID != 5 {
		t.Errorf("Expected varID 5, got %d", decoded.VarID)
	}
	if decoded.Properties["type"] != "Contact" {
		t.Error("Missing type property")
	}
	if decoded.Properties["inactive"] != "true" {
		t.Error("Missing inactive property")
	}
	if decoded.Value != nil {
		t.Error("Value should be nil for property-only update")
	}
}

// TestErrorMessageFormat verifies error message structure
func TestErrorMessageFormat(t *testing.T) {
	errMsg := ErrorMessage{
		VarID:       1,
		Code:        "INVALID_TYPE",
		Description: "Unknown presenter type 'Foo'",
	}

	msg, _ := NewMessage(MsgError, errMsg)
	encoded, _ := msg.Encode()

	parsed, _ := ParseMessage(encoded)
	if parsed.Type != MsgError {
		t.Errorf("Expected error type, got %s", parsed.Type)
	}

	var decoded ErrorMessage
	json.Unmarshal(parsed.Data, &decoded)

	if decoded.VarID != 1 {
		t.Errorf("Expected varID 1, got %d", decoded.VarID)
	}
	if decoded.Code != "INVALID_TYPE" {
		t.Errorf("Expected code INVALID_TYPE, got %s", decoded.Code)
	}
	if decoded.Description != "Unknown presenter type 'Foo'" {
		t.Errorf("Wrong description: %s", decoded.Description)
	}
}

// TestBatchIsBatchCheck verifies batch detection
func TestBatchIsBatchCheck(t *testing.T) {
	// Single message
	single := json.RawMessage(`{"type":"update","data":{"varId":1}}`)
	var msg Message
	err := json.Unmarshal(single, &msg)
	if err != nil {
		t.Log("Single message parses correctly")
	}

	// Batch (array)
	batch := json.RawMessage(`[{"type":"update","data":{"varId":1}},{"type":"update","data":{"varId":2}}]`)
	var msgs []Message
	err = json.Unmarshal(batch, &msgs)
	if err != nil {
		t.Errorf("Batch should parse as array: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("Expected 2 messages in batch, got %d", len(msgs))
	}
}

// TestEmptyBatcherFlushReturnsNil verifies empty flush behavior
func TestEmptyBatcherFlushReturnsNil(t *testing.T) {
	b := NewMessageBatcher()

	messages := b.Flush()
	if messages != nil {
		t.Error("Empty batcher Flush should return nil")
	}

	jsonBytes, err := b.FlushJSON()
	if err != nil {
		t.Errorf("FlushJSON error: %v", err)
	}
	if jsonBytes != nil {
		t.Error("Empty batcher FlushJSON should return nil")
	}
}
