// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

package xmpp

import (
	"encoding/xml"
)

// Message is an XMPP stanza that contains a payload for direct one-to-one
// communication with another network entity.  It is often used for sending chat
// messages to an individual or group chat server, or for notifications and
// alerts that don't require a response.
type Message struct {
	stanza

	XMLName xml.Name `xml:"message"`
}

type messageType int

const (
	// A NormalMessage is a standalone message that is sent outside the context of
	// a one-to-one conversation or groupchat, and to which it is expected that
	// the recipient will reply. Typically a receiving client will present a
	// message of type "normal" in an interface that enables the recipient to
	// reply, but without a conversation history.
	NormalMessage messageType = iota

	// ChatMessage represents a message sent in the context of a one-to-one chat
	// session.  Typically an interactive client will present a message of type
	// "chat" in an interface that enables one-to-one chat between the two
	// parties, including an appropriate conversation history.
	ChatMessage

	// An ErrorMessage is generated by an entity that experiences an error when
	// processing a message received from another entity.
	ErrorMessage

	// A GroupChatMessage is sent in the context of a multi-user chat environment.
	// Typically a receiving client will present a message of type "groupchat" in
	// an interface that enables many-to-many chat between the parties, including
	// a roster of parties in the chatroom and an appropriate conversation
	// history.
	GroupChatMessage

	// A HeadlineMessage provides an alert, a notification, or other transient
	// information to which no reply is expected (e.g., news headlines, sports
	// updates, near-real-time market data, or syndicated content). Because no
	// reply to the message is expected, typically a receiving client will present
	// a message of type "headline" in an interface that appropriately
	// differentiates the message from standalone messages, chat messages, and
	// groupchat messages (e.g., by not providing the recipient with the ability
	// to reply).
	HeadlineMessage
)