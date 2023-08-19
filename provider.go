// SPDX-FileCopyrightText: 2023 Steffen Vogel <post@steffenvogel.de>
// SPDX-License-Identifier: Apache-2.0

package token

type Secret [32]byte

type SecretProvider interface {
	Secret() (Secret, error)
}

type (
	PublicKey  []byte
	CipherText []byte
)

type SecretEncrypter interface {
	Encrypt(ss Secret, pk PublicKey) (ct CipherText, err error)
}

type SecretDecrypter interface {
	PublicKey() PublicKey
	Decrypt(ct CipherText) (ss Secret, err error)
}
