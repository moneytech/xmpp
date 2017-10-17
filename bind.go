// Copyright 2016 Sam Whited.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package xmpp

import (
	"context"
	"encoding/xml"
	"io"

	"mellium.im/xmlstream"
	"mellium.im/xmpp/internal"
	"mellium.im/xmpp/internal/ns"
	"mellium.im/xmpp/jid"
	"mellium.im/xmpp/stanza"
	"mellium.im/xmpp/stream"
)

// BindResource is a stream feature that can be used for binding a resource
// (the name by which an individual client can be addressed) to the stream.
//
// Resource binding is the final feature negotiated when setting up a new
// session and is required to allow communication with other clients and servers
// in the network. Resource binding is mandatory-to-negotiate.
//
// If used on a server connection, BindResource generates and assigns random
// resourceparts, however this default is subject to change.
func BindResource() StreamFeature {
	return bind(nil)
}

// BindCustom is identical to BindResource when used on a client session, but
// for server sessions the server function is called to generate the JID that
// should be returned to the client. If server is nil, BindCustom is identical
// to BindResource.
//
// The server function is passed the current client JID and the resource
// requested by the client (or an empty string if a specific resource was not
// requested). Resources generated by the server function should be random to
// prevent certain security issues related to guessing resourceparts.
func BindCustom(server func(*jid.JID, string) (*jid.JID, error)) StreamFeature {
	return bind(server)
}

type bindIQ struct {
	stanza.IQ

	Bind bindPayload   `xml:"urn:ietf:params:xml:ns:xmpp-bind bind,omitempty"`
	Err  *stanza.Error `xml:"error,ommitempty"`
}

func (biq *bindIQ) TokenReader() xml.TokenReader {
	if biq.Err != nil {
		return stanza.WrapIQ(biq.IQ.To, biq.IQ.Type, biq.Err.TokenReader())
	}

	return stanza.WrapIQ(biq.IQ.To, biq.IQ.Type, biq.Bind.TokenReader())
}

func (biq *bindIQ) WriteXML(w xmlstream.TokenWriter) (n int, err error) {
	return xmlstream.Copy(w, biq.TokenReader())
}

type bindPayload struct {
	Resource string   `xml:"resource,omitempty"`
	JID      *jid.JID `xml:"jid,omitempty"`
}

func (bp bindPayload) TokenReader() xml.TokenReader {
	if bp.JID != nil {
		return xmlstream.Wrap(
			xmlstream.ReaderFunc(func() (xml.Token, error) {
				return xml.CharData(bp.JID.String()), io.EOF
			}),
			xml.StartElement{Name: xml.Name{Local: "jid"}},
		)
	}

	return xmlstream.Wrap(
		xmlstream.ReaderFunc(func() (xml.Token, error) {
			return xml.CharData(bp.Resource), io.EOF
		}),
		xml.StartElement{Name: xml.Name{Local: "resource"}},
	)
}

func bind(server func(*jid.JID, string) (*jid.JID, error)) StreamFeature {
	return StreamFeature{
		Name:       xml.Name{Space: ns.Bind, Local: "bind"},
		Necessary:  Authn,
		Prohibited: Ready,
		List: func(ctx context.Context, e xmlstream.TokenWriter, start xml.StartElement) (req bool, err error) {
			req = true
			if err = e.EncodeToken(start); err != nil {
				return req, err
			}
			if err = e.EncodeToken(start.End()); err != nil {
				return req, err
			}

			return req, e.Flush()
		},
		Parse: func(ctx context.Context, r xml.TokenReader, start *xml.StartElement) (bool, interface{}, error) {
			parsed := struct {
				XMLName xml.Name `xml:"urn:ietf:params:xml:ns:xmpp-bind bind"`
			}{}
			return true, nil, xml.NewTokenDecoder(r).DecodeElement(&parsed, start)
		},
		Negotiate: func(ctx context.Context, session *Session, data interface{}) (mask SessionState, rw io.ReadWriter, err error) {
			d := xml.NewTokenDecoder(session)

			// Handle the server side of resource binding if we're on the receiving
			// end of the connection.
			if (session.State() & Received) == Received {
				tok, err := d.Token()
				if err != nil {
					return mask, nil, err
				}
				start, ok := tok.(xml.StartElement)
				if !ok {
					return mask, nil, stream.BadFormat
				}
				resReq := bindIQ{}
				switch start.Name {
				case xml.Name{Space: ns.Client, Local: "iq"}:
					if err = d.DecodeElement(&resReq, &start); err != nil {
						return mask, nil, err
					}
				default:
					return mask, nil, stream.BadFormat
				}

				iqid := internal.GetAttr(start.Attr, "id")

				var j *jid.JID
				if server != nil {
					j, err = server(session.RemoteAddr(), resReq.Bind.Resource)
				} else {
					j, err = session.RemoteAddr().WithResource(internal.RandomID())
				}
				stanzaErr, ok := err.(stanza.Error)
				if err != nil && !ok {
					return mask, nil, err
				}

				resp := bindIQ{
					IQ: stanza.IQ{
						ID:   iqid,
						From: resReq.IQ.To,
						To:   resReq.IQ.From,
						Type: stanza.ResultIQ,
					},
				}

				if ok {
					// If a stanza error was returned:
					resp.Err = &stanzaErr
				} else {

					resp.Bind = bindPayload{JID: j}
				}

				_, err = resp.WriteXML(session)
				return mask, nil, err
			}

			// Client encodes an IQ requesting resource binding.
			reqID := internal.RandomID()
			req := &bindIQ{
				IQ: stanza.IQ{
					ID:   reqID,
					Type: stanza.SetIQ,
				},
				Bind: bindPayload{
					Resource: session.origin.Resourcepart(),
				},
			}
			_, err = req.WriteXML(session)
			if err != nil {
				return mask, nil, err
			}

			// Client waits on an IQ response.
			//
			// We duplicate a lot of what should be stream-level IQ logic here; that
			// could maybe be fixed in the future, but it's necessary right now
			// because being able to use an IQ at all during resource negotiation is a
			// special case in XMPP that really shouldn't be valid (and is fixed in
			// current working drafts for a bind replacement).
			tok, err := d.Token()
			if err != nil {
				return mask, nil, err
			}
			start, ok := tok.(xml.StartElement)
			if !ok {
				return mask, nil, stream.BadFormat
			}
			resp := bindIQ{}
			switch start.Name {
			case xml.Name{Space: ns.Client, Local: "iq"}:
				if err = d.DecodeElement(&resp, &start); err != nil {
					return mask, nil, err
				}
			default:
				return mask, nil, stream.BadFormat
			}

			switch {
			case resp.ID != reqID:
				return mask, nil, stream.UndefinedCondition
			case resp.Type == stanza.ResultIQ:
				session.origin = resp.Bind.JID
			case resp.Type == stanza.ErrorIQ:
				return mask, nil, resp.Err
			default:
				return mask, nil, stanza.Error{Condition: stanza.BadRequest}
			}
			return Ready, nil, nil
		},
	}
}
