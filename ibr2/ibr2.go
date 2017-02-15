// Copyright 2017 Sam Whited.
// Use of this source code is governed by the BSD 2-clause license that can be
// found in the LICENSE file.

// Package ibr2 implements the Extensible In-Band Registration ProtoXEP.
package ibr2 // import "mellium.im/xmpp/ibr2"

import (
	"context"
	"encoding/xml"
	"errors"
	"io"

	"mellium.im/xmpp"
)

// Namespaces used by IBR.
const (
	NS = "urn:xmpp:register:0"
)

var (
	errNoChallenge = errors.New("No supported challenges were found")
)

func listFunc(challenges ...Challenge) func(context.Context, *xml.Encoder, xml.StartElement) (bool, error) {
	return func(ctx context.Context, e *xml.Encoder, start xml.StartElement) (req bool, err error) {
		if err = e.EncodeToken(start); err != nil {
			return
		}

		// List challenges
		seen := make(map[string]struct{})
		for _, c := range challenges {
			if _, ok := seen[c.Type]; ok {
				continue
			}
			challengeStart := xml.StartElement{
				Name: xml.Name{Local: "challenge"},
			}
			if err = e.EncodeToken(challengeStart); err != nil {
				return
			}
			if err = e.EncodeToken(xml.CharData(c.Type)); err != nil {
				return
			}
			if err = e.EncodeToken(challengeStart.End()); err != nil {
				return
			}
			seen[c.Type] = struct{}{}
		}

		if err = e.EncodeToken(start.End()); err != nil {
			return
		}
		return req, e.Flush()
	}
}

func parseFunc(challenges ...Challenge) func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (req bool, supported interface{}, err error) {
	return func(ctx context.Context, d *xml.Decoder, start *xml.StartElement) (bool, interface{}, error) {
		// Parse the list of challenge types sent down by the server.
		parsed := struct {
			Challenges []string `xml:"urn:xmpp:register:0 challenge"`
		}{}
		err := d.DecodeElement(&parsed, start)
		if err != nil {
			return false, false, err
		}

		// Dedup the lists of all challenge types supported by us and all challenge
		// types supported by the server.
		m := make(map[string]struct{})
		for _, c := range challenges {
			m[c.Type] = struct{}{}
		}
		for _, c := range parsed.Challenges {
			m[c] = struct{}{}
		}

		// If there are fewer types in the deduped aggregate list than in the
		// challenges we support, then the server list is a subset of the list we
		// support and we're okay to proceed with negotiation.
		return false, len(m) <= len(challenges), nil
	}
}

func negotiateFunc(challenges ...Challenge) func(context.Context, *xmpp.Session, interface{}) (xmpp.SessionState, io.ReadWriter, error) {
	return func(ctx context.Context, session *xmpp.Session, supported interface{}) (mask xmpp.SessionState, rw io.ReadWriter, err error) {
		server := (session.State() & xmpp.Received) == xmpp.Received

		if !server && !supported.(bool) {
			// We don't support some of the challenge types advertised by the server.
			// This is not an error, so don't return one; it just means we shouldn't
			// be negotiating this feature.
			return
		}

		// TODO:
		panic("not yet supported")
	}
}

// Register returns a new xmpp.StreamFeature that can be used to register a new
// account with the server.
func Register(challenges ...Challenge) xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name:       xml.Name{Local: "register", Space: NS},
		Necessary:  xmpp.Secure,
		Prohibited: xmpp.Authn,
		List:       listFunc(challenges...),
		Parse:      parseFunc(challenges...),
		Negotiate:  negotiateFunc(challenges...),
	}
}

// Recovery returns a new xmpp.StreamFeature that can be used to recover an
// account for which authentication credentials have been lost.
func Recovery(challenges ...Challenge) xmpp.StreamFeature {
	return xmpp.StreamFeature{
		Name:       xml.Name{Local: "recovery", Space: NS},
		Necessary:  xmpp.Secure,
		Prohibited: xmpp.Authn,
		List:       listFunc(challenges...),
		Parse:      parseFunc(challenges...),
		Negotiate:  negotiateFunc(challenges...),
	}
}