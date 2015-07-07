package otr3

import (
	"math/big"
	"testing"
)

func Test_generateSMPSecretGeneratesASecret(t *testing.T) {
	aliceFingerprint := hexToByte("0102030405060708090A0B0C0D0E0F1011121314")
	bobFingerprint := hexToByte("3132333435363738393A3B3C3D3E3F4041424344")
	ssid := hexToByte("FFF1D1E412345668")
	secret := []byte("this is something secret")
	result := generateSMPSecret(aliceFingerprint, bobFingerprint, ssid, secret)
	assertDeepEquals(t, result, hexToByte("D9B2E56321F9A9F8E364607C8C82DECD8E8E6209E2CB952C7E649620F5286FE3"))
}

func Test_generatesLongerAandRValuesForOtrV3(t *testing.T) {
	otr := context{otrV3{}, fixedRand([]string{
		"ABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		"BBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		"CBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		"DBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
	})}
	smp := otr.generateSMPStartParameters()
	assertDeepEquals(t, smp.a2, new(big.Int).SetBytes(hexToByte("ABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD")))
	assertDeepEquals(t, smp.a3, new(big.Int).SetBytes(hexToByte("BBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD")))
	assertDeepEquals(t, smp.r2, new(big.Int).SetBytes(hexToByte("CBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD")))
	assertDeepEquals(t, smp.r3, new(big.Int).SetBytes(hexToByte("DBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD")))
}

func Test_generatesShorterAandRValuesForOtrV2(t *testing.T) {
	otr := context{otrV2{}, fixedRand([]string{
		"ABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		"BBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		"CBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
		"DBCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCDABCD",
	})}
	smp := otr.generateSMPStartParameters()
	assertDeepEquals(t, smp.a2, new(big.Int).SetBytes(hexToByte("ABCDABCDABCDABCDABCDABCDABCDABCD")))
	assertDeepEquals(t, smp.a3, new(big.Int).SetBytes(hexToByte("BBCDABCDABCDABCDABCDABCDABCDABCD")))
	assertDeepEquals(t, smp.r2, new(big.Int).SetBytes(hexToByte("CBCDABCDABCDABCDABCDABCDABCDABCD")))
	assertDeepEquals(t, smp.r3, new(big.Int).SetBytes(hexToByte("DBCDABCDABCDABCDABCDABCDABCDABCD")))
}
