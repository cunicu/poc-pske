// SPDX-FileCopyrightText: 2023 Steffen Vogel
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"

	"github.com/ebfe/scard"

	"cunicu.li/hawkes/hmac/ykoath"
)

var _ PrivateKeyHMAC = (*ykoathKey)(nil)

const idChallenge = "hawkes/v1"

type ykoathKey struct {
	provider *ykoathProvider
	name     string
}

func (k *ykoathKey) ID() KeyID {
	id, _ := k.provider.keyID(k.name)
	return id
}

func (k *ykoathKey) Close() error {
	return nil
}

func (k *ykoathKey) Details() map[string]any {
	return map[string]any{}
}

func (k *ykoathKey) HMAC(chal []byte) ([]byte, error) {
	secret, _, err := k.provider.CalculateHOTP(k.name, chal)
	if err != nil {
		return nil, err
	}

	return secret, nil
}

var _ Provider = (*ykoathProvider)(nil)

type ykoathProvider struct {
	*ykoath.OATH
}

func newYKOATHProvider(card *scard.Card) (Provider, error) {
	oath, err := ykoath.New(card, 0)
	if err != nil {
		return nil, err
	}

	sel, err := oath.Select()
	if err != nil {
		return nil, fmt.Errorf("failed to select app: %w", err)
	}

	slog.Debug("Selected YKOATH applet",
		slog.String("version", fmt.Sprintf("%d.%d.%d", sel.Version[0], sel.Version[1], sel.Version[2])))

	return &ykoathProvider{
		OATH: oath,
	}, nil
}

func (p *ykoathProvider) Keys() (keyIDs []KeyID, err error) {
	slots, err := p.List()
	if err != nil {
		return nil, err
	}

	for _, slot := range slots {
		if slot.Algorithm != ykoath.HMACSHA256 {
			continue
		}

		keyID, err := p.keyID(slot.Name)
		if err != nil {
			return nil, err
		}

		keyIDs = append(keyIDs, keyID)
	}

	return keyIDs, nil
}

func (p *ykoathProvider) DestroyKey(id KeyID) error {
	name, err := p.nameByID(id)
	if err != nil {
		return err
	}

	return p.Delete(name)
}

func (p *ykoathProvider) CreateKeyFromSecret(label string, secret []byte) (KeyID, error) {
	if err := p.Put(label, ykoath.HMACSHA256, ykoath.TOTP, secret, false, 6); err != nil {
		return nil, err
	}

	id, err := p.keyID(label)
	if err != nil {
		return nil, err
	}

	return id, nil
}

func (p *ykoathProvider) CreateKey(label string) (KeyID, error) {
	secret := make([]byte, 20) // RFC4226 recommends a secret length of 160bits
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}

	return p.CreateKeyFromSecret(label, secret)
}

func (p *ykoathProvider) OpenKey(id KeyID) (PrivateKey, error) {
	name, err := p.nameByID(id)
	if err != nil {
		return nil, err
	}

	return &ykoathKey{
		provider: p,
		name:     name,
	}, nil
}

func (p *ykoathProvider) nameByID(id KeyID) (string, error) {
	slots, err := p.List()
	if err != nil {
		return "", err
	}

	for _, slot := range slots {
		if slot.Algorithm != ykoath.HMACSHA256 {
			continue
		}

		key, err := p.keyID(slot.Name)
		if err != nil {
			return "", err
		}

		if bytes.Equal(key, id) {
			return slot.Name, nil
		}
	}

	return "", os.ErrNotExist
}

func (p *ykoathProvider) keyID(name string) (KeyID, error) {
	key, _, err := p.CalculateHOTP(name, []byte(idChallenge))
	if err != nil {
		return nil, err
	}

	return key, nil
}

//nolint:gochecknoinits
func init() {
	Register("YKOATH", newYKOATHProvider)
}
