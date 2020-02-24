# Proposal: Implement stanza handlers

**Author(s):** Sam Whited  
**Last updated:** 2020-02-17  
**Status:** accepted  
**Discussion:** https://mellium.im/issue/25


## Abstract

An API is needed for multiplexing based on stanzas that allows for matching
based on type and quickly replying or re-marshaling the original stanzas.


## Background

In the previous [IQ mux proposal] a new mechanism for routing IQ stanzas based
on their type and payload was introduced.
In practice, registering the IQ muxer ended up being cumbersome, and the
previous proposal did not solve the problem of routing message or presence
stanzas (this was deliberately left for a future proposal).
To solve both of these problems at once the current [multiplexer] can be adapted
using what we learned from the IQ mux experiment such that it can route message
and presence stanzas based on their type, and IQs based on their type and
payload.

[IQ mux proposal]: https://mellium.im/design/18_iqmux
[multiplexer]: https://godoc.org/mellium.im/xmpp/mux#ServeMux


## Requirements

 - Ability to multiplex IQ stanzas by stanza type and payload name
 - Ability to multiplex message and presence stanzas by type
 - IQ, message, presence, and other general top level elements must have their
   own distinct handler types
 - The handlers must be extensible to add functionality such as replying to IQs
   or other thought of features in the future without making breaking API
   changes to the handler itself


## Proposal

In addition to the existing [`Handler`] and [`IQHandler`] types, the following
types would be added for messages and presence:

    type MessageHandler interface {
            HandleMessage(msg stanza.Message, t xmlstream.TokenReadEncoder) error
    }
        MessageHandler responds to message stanzas.

    type PresenceHandler interface {
            HandlePresence(p stanza.Presence, t xmlstream.TokenReadEncoder) error
    }
        PresenceHandler responds to presence stanzas.


Adapters for functions with the provided signature would also be made available,
similar to [`HandlerFunc`].

The existing IQ mux would be removed (this is a backwards incompatible change,
however, we are pre-1.0 and the IQ mux was never put into a release) and its
methods and functionality would be added to [`ServeMux`].

The options related to registering stanzas would then be modified to take the
new patterns as follows:

    func IQ(typ stanza.IQType, payload xml.Name, h IQHandler) Option
        IQ returns an option that matches IQ stanzas based on their type and the
        name of the payload.

    func Message(typ stanza.MessageType, h MessageHandler) Option
        Message returns an option that matches message stanzas by type.

    func Presence(typ stanza.PresenceType, h PresenceHandler) Option
        Presence returns an option that matches presence stanzas by type.

Functional versions of these options (taking a `HandlerFunc`) would also be
added.


[`Handler`]: https://godoc.org/mellium.im/xmpp#Handler
[`IQHandler`]: https://godoc.org/mellium.im/xmpp/mux#IQHandler
[`HandlerFunc`]: https://godoc.org/mellium.im/xmpp#HandlerFunc
[`ServeMux`]: https://godoc.org/mellium.im/xmpp/mux#ServeMux


## Open Questions

- What happens if the normal [`Handle`] options are used to register a handler
  that would match stanzas?
- We still need some way to match on Message/Presence payloads, otherwise how do
  we have different handlers for eg. IBB data payloads or messages with a body.


[`Handle`]: https://godoc.org/mellium.im/xmpp/mux#Handle