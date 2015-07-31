package otr3

import (
	"crypto/sha256"
	"math/big"
	"testing"
)

func dhMsgType(msg []byte) byte {
	return msg[2]
}

func dhMsgVersion(msg []byte) uint16 {
	_, protocolVersion, _ := extractShort(msg)
	return protocolVersion
}

func Test_conversationInitialState(t *testing.T) {
	c := newConversation(nil, fixtureRand())
	assertEquals(t, c.ake.state, authStateNone{})
}

func Test_receiveQueryMessage_SendDHCommitAndTransitToStateAwaitingDHKey(t *testing.T) {
	queryMsg := []byte("?OTRv3?")

	c := newConversation(nil, fixtureRand())
	c.policies.add(allowV3)
	msg, _ := c.receiveQueryMessage(queryMsg)

	assertEquals(t, c.ake.state, authStateAwaitingDHKey{})
	assertDeepEquals(t, fixtureDHCommitMsg(), msg)
}

func Test_receiveQueryMessageV2_SendDHCommitv2(t *testing.T) {
	queryMsg := []byte("?OTRv2?")

	c := newConversation(nil, fixtureRand())
	c.policies.add(allowV2)
	msg, _ := c.receiveQueryMessage(queryMsg)

	assertDeepEquals(t, fixtureDHCommitMsgV2(), msg)
}

func Test_receiveQueryMessage_StoresRAndXAndGx(t *testing.T) {
	fixture := fixtureConversation()
	fixture.dhCommitMessage()

	msg := []byte("?OTRv3?")
	cxt := newConversation(nil, fixtureRand())
	cxt.policies.add(allowV3)

	cxt.receiveQueryMessage(msg)
	assertDeepEquals(t, cxt.ake.r, fixture.ake.r)
	assertDeepEquals(t, cxt.ake.secretExponent, fixture.ake.secretExponent)
	assertDeepEquals(t, cxt.ake.ourPublicValue, fixture.ake.ourPublicValue)
}

func Test_parseOTRQueryMessage(t *testing.T) {
	var exp = map[string][]int{
		"?OTR?":     []int{1},
		"?OTRv2?":   []int{2},
		"?OTRv23?":  []int{2, 3},
		"?OTR?v2":   []int{1, 2},
		"?OTRv248?": []int{2, 4, 8},
		"?OTR?v?":   []int{1},
		"?OTRv?":    []int{},
	}

	for queryMsg, versions := range exp {
		m := []byte(queryMsg)
		assertDeepEquals(t, parseOTRQueryMessage(m), versions)
	}
}

func Test_acceptOTRRequest_returnsNilForUnsupportedVersions(t *testing.T) {
	p := policies(0)
	msg := []byte("?OTR?")
	v, ok := acceptOTRRequest(p, msg)

	assertEquals(t, v, nil)
	assertEquals(t, ok, false)
}

func Test_acceptOTRRequest_acceptsOTRV3IfHasAllowV3Policy(t *testing.T) {
	msg := []byte("?OTRv32?")
	p := policies(0)
	p.allowV2()
	p.allowV3()
	v, ok := acceptOTRRequest(p, msg)

	assertEquals(t, v, otrV3{})
	assertEquals(t, ok, true)
}

func Test_acceptOTRRequest_acceptsOTRV2IfHasOnlyAllowV2Policy(t *testing.T) {
	msg := []byte("?OTRv32?")
	p := policies(0)
	p.allowV2()
	v, ok := acceptOTRRequest(p, msg)

	assertEquals(t, v, otrV2{})
	assertEquals(t, ok, true)
}

func Test_receiveDHCommit_TransitionsFromNoneToAwaitingRevealSigAndSendDHKeyMsg(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())
	nextState, nextMsg, e := authStateNone{}.receiveDHCommitMessage(c, fixtureDHCommitMsg())

	assertEquals(t, nextState, authStateAwaitingRevealSig{})
	assertEquals(t, dhMsgType(nextMsg), msgTypeDHKey)
	assertEquals(t, e, nil)
}

func Test_receiveDHCommit_AtAuthStateNoneStoresGyAndY(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())
	authStateNone{}.receiveDHCommitMessage(c, fixtureDHCommitMsg())

	assertDeepEquals(t, c.ake.ourPublicValue, fixedgy)
	assertDeepEquals(t, c.ake.secretExponent, fixedy)
}

func Test_receiveDHCommit_AtAuthStateNoneStoresEncryptedGxAndHashedGx(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())

	dhCommitMsg := fixtureDHCommitMsg()
	newMsg, encryptedGx, _ := extractData(dhCommitMsg[otrv3HeaderLen:])
	_, hashedGx, _ := extractData(newMsg)

	authStateNone{}.receiveDHCommitMessage(c, dhCommitMsg)

	assertDeepEquals(t, c.ake.hashedGx[:], hashedGx)
	assertDeepEquals(t, c.ake.encryptedGx, encryptedGx)
}

func Test_receiveDHCommit_ResendPreviousDHKeyMsgFromAwaitingRevealSig(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())

	authAwaitingRevSig, prevDHKeyMsg, _ := authStateNone{}.receiveDHCommitMessage(c, fixtureDHCommitMsg())
	assertEquals(t, authAwaitingRevSig, authStateAwaitingRevealSig{})

	nextState, msg, _ := authAwaitingRevSig.receiveDHCommitMessage(c, fixtureDHCommitMsg())

	assertEquals(t, nextState, authStateAwaitingRevealSig{})
	assertEquals(t, dhMsgType(msg), msgTypeDHKey)
	assertDeepEquals(t, prevDHKeyMsg, msg)
}

func Test_receiveDHCommit_AtAuthAwaitingRevealSigiForgetOldEncryptedGxAndHashedGx(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	//TODO needs to stores encryptedGx and hashedGx when it is generated
	c.ake.encryptedGx = []byte{0x02}         //some encryptedGx
	c.ake.hashedGx = [sha256.Size]byte{0x05} //some hashedGx

	newDHCommitMsg := fixtureDHCommitMsg()
	newMsg, newEncryptedGx, _ := extractData(newDHCommitMsg[otrv3HeaderLen:])
	_, newHashedGx, _ := extractData(newMsg)

	authStateNone{}.receiveDHCommitMessage(c, fixtureDHCommitMsg())

	authStateAwaitingRevealSig{}.receiveDHCommitMessage(c, newDHCommitMsg)
	assertDeepEquals(t, c.ake.encryptedGx, newEncryptedGx)
	assertDeepEquals(t, c.ake.hashedGx[:], newHashedGx)
}

func Test_receiveDHCommit_AtAuthAwaitingSigTransitionsToAwaitingRevSigAndSendsNewDHKeyMsg(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())

	authAwaitingRevSig, msg, _ := authStateAwaitingSig{}.receiveDHCommitMessage(c, fixtureDHCommitMsg())
	assertEquals(t, authAwaitingRevSig, authStateAwaitingRevealSig{})
	assertEquals(t, dhMsgType(msg), msgTypeDHKey)
}

func Test_receiveDHCommit_AtAwaitingDHKeyIgnoreIncomingMsgAndResendOurDHCommitMsgIfOurHashIsHigher(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHMsg, _ := ourDHCommitAKE.dhCommitMessage()

	//make sure we store the same values when creating the DH commit
	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.ake.encryptedGx = ourDHCommitAKE.ake.encryptedGx
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	// force their hashedGx to be lower than ours
	msg := fixtureDHCommitMsg()
	newPoint, _, _ := extractData(msg[otrv3HeaderLen:])
	newPoint[4] = 0x00

	state, newMsg, err := authStateAwaitingDHKey{}.receiveDHCommitMessage(c, msg)
	assertDeepEquals(t, err, nil)
	assertEquals(t, state, authStateAwaitingRevealSig{})
	assertDeepEquals(t, newMsg[otrv3HeaderLen:], ourDHMsg[otrv3HeaderLen:])
}

func Test_receiveDHCommit_AtAwaitingDHKeyForgetOurGxAndSendDHKeyMsgAndGoToAwaitingRevealSig(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	//make sure we store the same values when creating the DH commit
	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	// force their hashedGx to be higher than ours
	msg := fixtureDHCommitMsg()
	newPoint, _, _ := extractData(msg[otrv3HeaderLen:])
	newPoint[4] = 0xFF

	state, newMsg, _ := authStateAwaitingDHKey{}.receiveDHCommitMessage(c, msg)
	assertEquals(t, state, authStateAwaitingRevealSig{})
	assertEquals(t, dhMsgType(newMsg), msgTypeDHKey)
	assertDeepEquals(t, c.ake.ourPublicValue, fixedgy)
	assertDeepEquals(t, c.ake.secretExponent, fixedy)
}

func Test_receiveDHKey_AtAuthStateNoneOrAuthStateAwaitingRevealSigIgnoreIt(t *testing.T) {
	var nilB []byte
	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	dhKeymsg := fixtureDHKeyMsg(otrV3{})

	states := []authState{
		authStateNone{},
		authStateAwaitingRevealSig{},
	}

	for _, s := range states {
		state, msg, err := s.receiveDHKeyMessage(c, dhKeymsg)
		assertEquals(t, err, nil)
		assertEquals(t, state, s)
		assertDeepEquals(t, msg, nilB)
	}
}

func Test_receiveDHKey_TransitionsFromAwaitingDHKeyToAwaitingSigAndSendsRevealSig(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := bobContextAtAwaitingDHKey()

	state, msg, _ := authStateAwaitingDHKey{}.receiveDHKeyMessage(c, fixtureDHKeyMsg(otrV3{}))

	//TODO before generate rev si need to extract their gy from DH commit
	_, ok := state.(authStateAwaitingSig)
	assertEquals(t, ok, true)
	assertEquals(t, dhMsgType(msg), msgTypeRevealSig)
}

func Test_receiveDHKey_AtAwaitingDHKeyStoresGyAndSigKey(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := bobContextAtAwaitingDHKey()

	_, _, err := authStateAwaitingDHKey{}.receiveDHKeyMessage(c, fixtureDHKeyMsg(otrV3{}))

	assertEquals(t, err, nil)
	assertDeepEquals(t, c.ake.theirPublicValue, fixedgy)
	assertDeepEquals(t, c.ake.sigKey.c[:], expectedC)
	assertDeepEquals(t, c.ake.sigKey.m1[:], expectedM1)
	assertDeepEquals(t, c.ake.sigKey.m2[:], expectedM2)
}

func Test_receiveDHKey_AtAwaitingDHKeyStoresOursAndTheirDHKeysAndIncreaseCounter(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := bobContextAtAwaitingDHKey()

	_, _, err := authStateAwaitingDHKey{}.receiveDHKeyMessage(c, fixtureDHKeyMsg(otrV3{}))

	assertEquals(t, err, nil)
	assertDeepEquals(t, c.keys.theirCurrentDHPubKey, fixedgy)
	assertDeepEquals(t, c.keys.ourCurrentDHKeys.pub, fixedgx)
	assertDeepEquals(t, c.keys.ourCurrentDHKeys.priv, fixedx)
	assertEquals(t, c.keys.ourCounter, uint64(1))
	assertEquals(t, c.keys.ourKeyID, uint32(1))
	assertEquals(t, c.keys.theirKeyID, uint32(0))
}

func Test_receiveDHKey_AtAuthAwaitingSigIfReceivesSameDHKeyMsgRetransmitRevealSigMsg(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.setSecretExponent(ourDHCommitAKE.ake.secretExponent)
	c.ourKey = bobPrivateKey

	sameDHKeyMsg := fixtureDHKeyMsg(otrV3{})
	sigState, previousRevealSig, _ := authStateAwaitingDHKey{}.receiveDHKeyMessage(c, sameDHKeyMsg)

	state, msg, _ := sigState.receiveDHKeyMessage(c, sameDHKeyMsg)

	//FIXME: What about gy and sigKey?
	_, sameStateType := state.(authStateAwaitingSig)
	assertDeepEquals(t, sameStateType, true)
	assertDeepEquals(t, msg, previousRevealSig)
}

func Test_receiveDHKey_AtAuthAwaitingSigIgnoresMsgIfIsNotSameDHKeyMsg(t *testing.T) {
	var nilB []byte

	newDHKeyMsg := fixtureDHKeyMsg(otrV3{})
	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()

	state, msg, _ := authStateAwaitingSig{}.receiveDHKeyMessage(c, newDHKeyMsg)

	_, sameStateType := state.(authStateAwaitingSig)
	assertDeepEquals(t, sameStateType, true)
	assertDeepEquals(t, msg, nilB)
}

func Test_receiveRevealSig_TransitionsFromAwaitingRevealSigToNoneOnSuccess(t *testing.T) {
	revealSignMsg := fixtureRevealSigMsg(otrV2{})

	c := aliceContextAtAwaitingRevealSig()

	state, msg, err := authStateAwaitingRevealSig{}.receiveRevealSigMessage(c, revealSignMsg)

	assertEquals(t, err, nil)
	assertEquals(t, state, authStateNone{})
	assertEquals(t, dhMsgType(msg), msgTypeSig)
}

func Test_receiveRevealSig_AtAwaitingRevealSigStoresOursAndTheirDHKeysAndIncreaseCounter(t *testing.T) {
	var nilBigInt *big.Int
	revealSignMsg := fixtureRevealSigMsg(otrV2{})

	c := aliceContextAtAwaitingRevealSig()

	_, _, err := authStateAwaitingRevealSig{}.receiveRevealSigMessage(c, revealSignMsg)

	assertEquals(t, err, nil)
	assertDeepEquals(t, c.keys.theirCurrentDHPubKey, fixedgx)
	assertDeepEquals(t, c.keys.theirPreviousDHPubKey, nilBigInt)
	assertDeepEquals(t, c.keys.ourCurrentDHKeys.pub, fixedgy)
	assertDeepEquals(t, c.keys.ourCurrentDHKeys.priv, fixedy)
	assertEquals(t, c.keys.ourCounter, uint64(1))
	assertEquals(t, c.keys.ourKeyID, uint32(1))
	assertEquals(t, c.keys.theirKeyID, uint32(1))
}

func Test_authStateAwaitingRevealSig_receiveRevealSigMessage_returnsErrorIfProcessRevealSigFails(t *testing.T) {
	c := newConversation(otrV2{}, fixtureRand())
	c.policies.add(allowV2)
	_, _, err := authStateAwaitingRevealSig{}.receiveRevealSigMessage(c, []byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_receiveRevealSig_IgnoreMessageIfNotInStateAwaitingRevealSig(t *testing.T) {
	var nilB []byte

	states := []authState{
		authStateNone{},
		authStateAwaitingDHKey{},
		authStateAwaitingSig{},
	}

	revealSignMsg := fixtureRevealSigMsg(otrV2{})

	for _, s := range states {
		c := newConversation(otrV3{}, fixtureRand())
		state, msg, err := s.receiveRevealSigMessage(c, revealSignMsg)

		assertEquals(t, err, nil)
		assertDeepEquals(t, state, s)
		assertDeepEquals(t, msg, nilB)
	}
}

func Test_receiveSig_TransitionsFromAwaitingSigToNoneOnSuccess(t *testing.T) {
	var nilB []byte
	sigMsg := fixtureSigMsg(otrV2{})
	c := bobContextAtAwaitingSig()

	state, msg, err := authStateAwaitingSig{}.receiveSigMessage(c, sigMsg)

	assertEquals(t, err, nil)
	assertEquals(t, state, authStateNone{})
	assertDeepEquals(t, msg, nilB)
	assertEquals(t, c.keys.theirKeyID, uint32(1))
}

func Test_receiveSig_IgnoreMessageIfNotInStateAwaitingSig(t *testing.T) {
	var nilB []byte

	states := []authState{
		authStateNone{},
		authStateAwaitingDHKey{},
		authStateAwaitingRevealSig{},
	}

	revealSignMsg := fixtureRevealSigMsg(otrV2{})

	for _, s := range states {
		c := newConversation(otrV3{}, fixtureRand())
		state, msg, err := s.receiveSigMessage(c, revealSignMsg)

		assertEquals(t, err, nil)
		assertEquals(t, state, s)
		assertDeepEquals(t, msg, nilB)
	}
}

func Test_receiveAKE_ignoresDHCommitIfItsVersionIsNotInThePolicy(t *testing.T) {
	var nilB []byte
	cV2 := newConversation(otrV2{}, fixtureRand())
	cV2.policies.add(allowV2)

	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.policies.add(allowV3)

	ake := fixtureConversationV2()
	msgV2, _ := ake.dhCommitMessage()
	msgV3 := fixtureDHCommitMsg()

	toSend, _ := cV2.receiveAKE(msgV3)
	assertEquals(t, cV2.ake.state, authStateNone{})
	assertDeepEquals(t, toSend, nilB)

	toSend, _ = cV3.receiveAKE(msgV2)
	assertEquals(t, cV3.ake.state, authStateNone{})
	assertDeepEquals(t, toSend, nilB)
}

func Test_receiveAKE_resolveProtocolVersionFromDHCommitMessage(t *testing.T) {
	c := newConversation(nil, fixtureRand())
	c.policies = policies(allowV3)
	c.receiveAKE(fixtureDHCommitMsg())

	assertEquals(t, c.version, otrV3{})

	c.policies = policies(allowV2)
	c.receiveAKE(fixtureDHCommitMsgV2())

	assertEquals(t, c.version, otrV2{})
}

func Test_receiveAKE_ignoresDHKeyIfItsVersionIsNotInThePolicy(t *testing.T) {
	var nilB []byte
	cV2 := newConversation(otrV2{}, fixtureRand())
	cV2.ensureAKE()
	cV2.ake.state = authStateAwaitingDHKey{}
	cV2.policies.add(allowV2)

	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.ensureAKE()
	cV3.ake.state = authStateAwaitingDHKey{}
	cV3.policies.add(allowV3)

	msgV2 := fixtureDHKeyMsg(otrV2{})
	msgV3 := fixtureDHKeyMsg(otrV3{})

	toSend, _ := cV2.receiveAKE(msgV3)
	assertEquals(t, cV2.ake.state, authStateAwaitingDHKey{})
	assertDeepEquals(t, toSend, nilB)

	toSend, _ = cV3.receiveAKE(msgV2)
	assertEquals(t, cV3.ake.state, authStateAwaitingDHKey{})
	assertDeepEquals(t, toSend, nilB)
}

func Test_receiveAKE_ignoresRevealSigIfItsVersionIsNotInThePolicy(t *testing.T) {
	var nilB []byte
	cV2 := newConversation(otrV2{}, fixtureRand())
	cV2.ensureAKE()
	cV2.ake.state = authStateAwaitingRevealSig{}
	cV2.policies.add(allowV2)

	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.ensureAKE()
	cV3.ake.state = authStateAwaitingRevealSig{}
	cV3.policies.add(allowV3)

	msgV2 := fixtureRevealSigMsg(otrV2{})
	msgV3 := fixtureRevealSigMsg(otrV3{})

	toSend, _ := cV2.receiveAKE(msgV3)
	assertEquals(t, cV2.ake.state, authStateAwaitingRevealSig{})
	assertDeepEquals(t, toSend, nilB)

	toSend, _ = cV3.receiveAKE(msgV2)
	assertEquals(t, cV3.ake.state, authStateAwaitingRevealSig{})
	assertDeepEquals(t, toSend, nilB)
}

func Test_receiveAKE_ignoresSignatureIfItsVersionIsNotInThePolicy(t *testing.T) {
	var nilB []byte
	cV2 := newConversation(otrV2{}, fixtureRand())
	cV2.ensureAKE()
	cV2.ake.state = authStateAwaitingSig{}
	cV2.policies.add(allowV2)

	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.ensureAKE()
	cV3.ake.state = authStateAwaitingSig{}
	cV3.policies.add(allowV3)

	msgV2 := fixtureSigMsg(otrV2{})
	msgV3 := fixtureSigMsg(otrV3{})

	toSend, _ := cV2.receiveAKE(msgV3)
	_, sameStateType := cV2.ake.state.(authStateAwaitingSig)
	assertDeepEquals(t, sameStateType, true)
	assertDeepEquals(t, toSend, nilB)

	toSend, _ = cV3.receiveAKE(msgV2)
	_, sameStateType = cV3.ake.state.(authStateAwaitingSig)
	assertDeepEquals(t, sameStateType, true)
	assertDeepEquals(t, toSend, nilB)
}

func Test_receiveAKE_ignoresRevealSignaureIfDoesNotAllowV2(t *testing.T) {
	var nilB []byte
	cV2 := newConversation(otrV2{}, fixtureRand())
	cV2.ensureAKE()
	cV2.ake.state = authStateAwaitingRevealSig{}
	cV2.policies = policies(allowV3)

	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.ensureAKE()
	cV3.ake.state = authStateAwaitingRevealSig{}
	cV3.policies = policies(allowV3)

	msgV2 := fixtureRevealSigMsg(otrV2{})
	msgV3 := fixtureRevealSigMsg(otrV3{})

	toSend, _ := cV3.receiveAKE(msgV3)
	assertEquals(t, cV3.ake.state, authStateAwaitingRevealSig{})
	assertDeepEquals(t, toSend, nilB)

	toSend, _ = cV2.receiveAKE(msgV2)
	assertEquals(t, cV2.ake.state, authStateAwaitingRevealSig{})
	assertDeepEquals(t, toSend, nilB)
}

func Test_receiveAKE_ignoresSignatureIfDoesNotAllowV2(t *testing.T) {
	var nilB []byte
	cV2 := newConversation(otrV2{}, fixtureRand())
	cV2.ensureAKE()
	cV2.ake.state = authStateAwaitingSig{}
	cV2.policies = policies(allowV3)

	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.ensureAKE()
	cV3.ake.state = authStateAwaitingSig{}
	cV3.policies = policies(allowV3)

	msgV2 := fixtureSigMsg(otrV2{})
	msgV3 := fixtureSigMsg(otrV3{})

	toSend, _ := cV3.receiveAKE(msgV3)

	assertDeepEquals(t, cV3.ake.state, authStateAwaitingSig{})
	assertDeepEquals(t, toSend, nilB)
	toSend, _ = cV2.receiveAKE(msgV2)
	assertDeepEquals(t, cV2.ake.state, authStateAwaitingSig{})
	assertDeepEquals(t, toSend, nilB)
}

func Test_receiveAKE_returnsErrorIfTheMessageIsCorrupt(t *testing.T) {
	cV3 := newConversation(otrV3{}, fixtureRand())
	cV3.ensureAKE()
	cV3.ake.state = authStateAwaitingSig{}
	cV3.policies.add(allowV3)

	_, err := cV3.receiveAKE([]byte{})
	assertEquals(t, err, errInvalidOTRMessage)

	_, err = cV3.receiveAKE([]byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)

	_, err = cV3.receiveAKE([]byte{0x00, 0x03, 0x56})
	assertDeepEquals(t, err, newOtrError("unknown message type 0x56"))
}

func Test_receiveAKE_receiveRevealSigMessageAndSetMessageStateToEncrypted(t *testing.T) {
	c := aliceContextAtAwaitingRevealSig()
	msg := fixtureRevealSigMsg(otrV2{})
	assertEquals(t, c.msgState, plainText)

	_, err := c.receiveAKE(msg)

	assertEquals(t, err, nil)
	assertEquals(t, c.msgState, encrypted)
}

func Test_receiveAKE_receiveRevealSigMessageAndStoresTheirKeyIDAndTheirCurrentDHPubKey(t *testing.T) {
	var nilBigInt *big.Int

	c := aliceContextAtAwaitingRevealSig()
	msg := fixtureRevealSigMsg(otrV2{})
	assertEquals(t, c.msgState, plainText)

	_, err := c.receiveAKE(msg)

	assertEquals(t, err, nil)
	assertEquals(t, c.keys.theirKeyID, uint32(1))
	assertDeepEquals(t, c.keys.theirCurrentDHPubKey, fixedgx)
	assertEquals(t, c.keys.theirPreviousDHPubKey, nilBigInt)
}

func Test_receiveAKE_receiveSigMessageAndSetMessageStateToEncrypted(t *testing.T) {
	c := bobContextAtAwaitingSig()
	msg := fixtureSigMsg(otrV2{})
	assertEquals(t, c.msgState, plainText)

	_, err := c.receiveAKE(msg)

	assertEquals(t, err, nil)
	assertEquals(t, c.msgState, encrypted)
}

func Test_receiveAKE_receiveSigMessageAndStoresTheirKeyIDAndTheirCurrentDHPubKey(t *testing.T) {
	var nilBigInt *big.Int

	c := bobContextAtAwaitingSig()

	msg := fixtureSigMsg(otrV2{})
	assertEquals(t, c.msgState, plainText)

	_, err := c.receiveAKE(msg)

	assertEquals(t, err, nil)
	assertEquals(t, c.keys.theirKeyID, uint32(1))
	assertDeepEquals(t, c.keys.theirCurrentDHPubKey, fixedgy)
	assertEquals(t, c.keys.theirPreviousDHPubKey, nilBigInt)
}

func Test_authStateAwaitingDHKey_receiveDHKeyMessage_returnsErrorIfprocessDHKeyReturnsError(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.setSecretExponent(ourDHCommitAKE.ake.secretExponent)
	c.ourKey = bobPrivateKey

	_, _, err := authStateAwaitingDHKey{}.receiveDHKeyMessage(c, []byte{0x01, 0x02})

	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateAwaitingDHKey_receiveDHKeyMessage_returnsErrorIfrevealSigMessageReturnsError(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV3{}, fixedRand([]string{"ABCD"}))
	c.startAKE()
	c.setSecretExponent(ourDHCommitAKE.ake.secretExponent)
	c.ourKey = bobPrivateKey

	sameDHKeyMsg := fixtureDHKeyMsg(otrV3{})
	_, _, err := authStateAwaitingDHKey{}.receiveDHKeyMessage(c, sameDHKeyMsg)

	assertEquals(t, err, errShortRandomRead)
}

func Test_authStateAwaitingSig_receiveDHKeyMessage_returnsErrorIfprocessDHKeyReturnsError(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.setSecretExponent(ourDHCommitAKE.ake.secretExponent)
	c.ourKey = bobPrivateKey

	_, _, err := authStateAwaitingSig{}.receiveDHKeyMessage(c, []byte{0x01, 0x02})

	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateAwaitingSig_receiveSigMessage_returnsErrorIfProcessSigFails(t *testing.T) {
	c := newConversation(otrV2{}, fixtureRand())
	c.policies.add(allowV2)
	_, _, err := authStateAwaitingSig{}.receiveSigMessage(c, []byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateAwaitingRevealSig_receiveDHCommitMessage_returnsErrorIfProcessDHCommitOrGenerateCommitInstanceTagsFailsFails(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateAwaitingRevealSig{}.receiveDHCommitMessage(c, []byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateNone_receiveDHCommitMessage_returnsErrorIfgenerateCommitMsgInstanceTagsFails(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV3{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateNone{}.receiveDHCommitMessage(c, []byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateNone_receiveDHCommitMessage_returnsErrorIfdhKeyMessageFails(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV2{}, fixedRand([]string{"ABCD"}))
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateNone{}.receiveDHCommitMessage(c, []byte{0x00, 0x00, 0x00})
	assertEquals(t, err, errShortRandomRead)
}

func Test_authStateNone_receiveDHCommitMessage_returnsErrorIfProcessDHCommitFails(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV2{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateNone{}.receiveDHCommitMessage(c, []byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateNone_receiveQueryMessage_returnsNoErrorForValidMessage(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())
	c.policies.add(allowV3)
	_, err := c.receiveQueryMessage([]byte("?OTRv3?"))
	assertEquals(t, err, nil)
}

func Test_authStateNone_receiveQueryMessage_returnsErrorIfNoCompatibleVersionCouldBeFound(t *testing.T) {
	c := newConversation(otrV3{}, fixtureRand())
	c.policies.add(allowV3)
	_, err := c.receiveQueryMessage([]byte("?OTRv2?"))
	assertEquals(t, err, errInvalidVersion)
}

func Test_authStateNone_receiveQueryMessage_returnsErrorIfDhCommitMessageGeneratesError(t *testing.T) {
	c := newConversation(otrV2{}, fixedRand([]string{"ABCDABCD"}))
	c.policies.add(allowV2)
	_, err := c.receiveQueryMessage([]byte("?OTRv2?"))
	assertEquals(t, err, errShortRandomRead)
}

func Test_authStateAwaitingDHKey_receiveDHCommitMessage_failsIfMsgDoesntHaveHeader(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV2{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateAwaitingDHKey{}.receiveDHCommitMessage(c, []byte{0x00, 0x00})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateAwaitingDHKey_receiveDHCommitMessage_failsIfCantExtractFirstPart(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV2{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateAwaitingDHKey{}.receiveDHCommitMessage(c, []byte{0x00, 0x00, 0x00, 0x01})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateAwaitingDHKey_receiveDHCommitMessage_failsIfCantExtractSecondPart(t *testing.T) {
	ourDHCommitAKE := fixtureConversation()
	ourDHCommitAKE.dhCommitMessage()

	c := newConversation(otrV2{}, fixtureRand())
	c.startAKE()
	c.ake.theirPublicValue = ourDHCommitAKE.ake.ourPublicValue

	_, _, err := authStateAwaitingDHKey{}.receiveDHCommitMessage(c, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x01, 0x00, 0x01, 0x02})
	assertEquals(t, err, errInvalidOTRMessage)
}

func Test_authStateNone_String_returnsTheCorrectString(t *testing.T) {
	assertEquals(t, authStateNone{}.String(), "AUTHSTATE_NONE")
}

func Test_authStateAwaitingDHKey_String_returnsTheCorrectString(t *testing.T) {
	assertEquals(t, authStateAwaitingDHKey{}.String(), "AUTHSTATE_AWAITING_DHKEY")
}

func Test_authStateAwaitingRevealSig_String_returnsTheCorrectString(t *testing.T) {
	assertEquals(t, authStateAwaitingRevealSig{}.String(), "AUTHSTATE_AWAITING_REVEALSIG")
}

func Test_authStateAwaitingSig_String_returnsTheCorrectString(t *testing.T) {
	assertEquals(t, authStateAwaitingSig{}.String(), "AUTHSTATE_AWAITING_SIG")
}

func Test_authStateV1Setup_String_returnsTheCorrectString(t *testing.T) {
	assertEquals(t, authStateV1Setup{}.String(), "AUTHSTATE_V1_SETUP")
}
