package drckks

import (
	"flag"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ldsec/lattigo/v2/rckks"
	"github.com/ldsec/lattigo/v2/ring"
	"github.com/ldsec/lattigo/v2/utils"
)

var flagLongTest = flag.Bool("long", false, "run the long test suite (all parameters). Overrides -short and requires -timeout=0.")
var printPrecisionStats = flag.Bool("print-precision", false, "print precision stats")
var minPrec float64 = 15.0
var parties uint64 = 3

func testString(opname string, parties uint64, params *rckks.Parameters) string {
	return fmt.Sprintf("%sparties=%d/logN=%d/logQ=%d/levels=%d/a=%d/b=%d",
		opname,
		parties,
		params.LogN(),
		params.LogQP(),
		params.MaxLevel()+1,
		params.Alpha(),
		params.Beta())
}

type testContext struct {
	params *rckks.Parameters

	drckksContext *drckksContext

	prng utils.PRNG

	encoder   rckks.Encoder
	evaluator rckks.Evaluator

	encryptorPk0 rckks.Encryptor
	decryptorSk0 rckks.Decryptor
	decryptorSk1 rckks.Decryptor

	pk0 *rckks.PublicKey
	pk1 *rckks.PublicKey

	sk0 *rckks.SecretKey
	sk1 *rckks.SecretKey

	sk0Shards []*rckks.SecretKey
	sk1Shards []*rckks.SecretKey
}

func TestDCKKS(t *testing.T) {

	rand.Seed(time.Now().UnixNano())

	var err error
	var testCtx = new(testContext)

	var defaultParams = rckks.DefaultParams[rckks.PN12QP109 : rckks.PN12QP109+4] // the default test runs for ring degree N=2^12, 2^13, 2^14, 2^15
	if testing.Short() {
		defaultParams = rckks.DefaultParams[rckks.PN12QP109 : rckks.PN12QP109+2] // the short test runs for ring degree N=2^12, 2^13
	}
	if *flagLongTest {
		defaultParams = rckks.DefaultParams // the long test suite runs for all default parameters
	}

	for _, p := range defaultParams {

		if testCtx, err = genTestParams(p); err != nil {
			panic(err)
		}

		testPublicKeyGen(testCtx, t)
		testRelinKeyGen(testCtx, t)
		testRotKeyGenCols(testCtx, t)
	}
}

func genTestParams(defaultParams *rckks.Parameters) (testCtx *testContext, err error) {

	testCtx = new(testContext)

	testCtx.params = defaultParams.Copy()

	testCtx.drckksContext = newDrckksContext(testCtx.params)

	if testCtx.prng, err = utils.NewPRNG(); err != nil {
		return nil, err
	}

	testCtx.encoder = rckks.NewEncoder(testCtx.params)
	testCtx.evaluator = rckks.NewEvaluator(testCtx.params)

	kgen := rckks.NewKeyGenerator(testCtx.params)

	// SecretKeys
	testCtx.sk0Shards = make([]*rckks.SecretKey, parties)
	testCtx.sk1Shards = make([]*rckks.SecretKey, parties)
	tmp0 := testCtx.drckksContext.ringQP.NewPoly()
	tmp1 := testCtx.drckksContext.ringQP.NewPoly()

	for j := uint64(0); j < parties; j++ {
		testCtx.sk0Shards[j] = kgen.GenSecretKey()
		testCtx.sk1Shards[j] = kgen.GenSecretKey()
		testCtx.drckksContext.ringQP.Add(tmp0, testCtx.sk0Shards[j].Get(), tmp0)
		testCtx.drckksContext.ringQP.Add(tmp1, testCtx.sk1Shards[j].Get(), tmp1)
	}

	testCtx.sk0 = new(rckks.SecretKey)
	testCtx.sk1 = new(rckks.SecretKey)

	testCtx.sk0.Set(tmp0)
	testCtx.sk1.Set(tmp1)

	// Publickeys
	testCtx.pk0 = kgen.GenPublicKey(testCtx.sk0)
	testCtx.pk1 = kgen.GenPublicKey(testCtx.sk1)

	testCtx.encryptorPk0 = rckks.NewEncryptorFromPk(testCtx.params, testCtx.pk0)
	testCtx.decryptorSk0 = rckks.NewDecryptor(testCtx.params, testCtx.sk0)
	testCtx.decryptorSk1 = rckks.NewDecryptor(testCtx.params, testCtx.sk1)

	return
}

func testPublicKeyGen(testCtx *testContext, t *testing.T) {

	decryptorSk0 := testCtx.decryptorSk0
	sk0Shards := testCtx.sk0Shards

	crpGenerator := ring.NewUniformSampler(testCtx.prng, testCtx.drckksContext.ringQP)

	t.Run(testString("PublicKeyGen/", parties, testCtx.params), func(t *testing.T) {

		crp := crpGenerator.ReadNew()

		type Party struct {
			*CKGProtocol
			s  *ring.Poly
			s1 CKGShare
		}

		ckgParties := make([]*Party, parties)
		for i := uint64(0); i < parties; i++ {
			p := new(Party)
			p.CKGProtocol = NewCKGProtocol(testCtx.params)
			p.s = sk0Shards[i].Get()
			p.s1 = p.AllocateShares()
			ckgParties[i] = p
		}
		P0 := ckgParties[0]

		// Each party creates a new CKGProtocol instance
		for i, p := range ckgParties {
			p.GenShare(p.s, crp, p.s1)
			if i > 0 {
				P0.AggregateShares(p.s1, P0.s1, P0.s1)
			}
		}

		pk := &rckks.PublicKey{}
		P0.GenPublicKey(P0.s1, crp, pk)

		// Verifies that decrypt((encryptp(collectiveSk, m), collectivePk) = m
		encryptorTest := rckks.NewEncryptorFromPk(testCtx.params, pk)

		coeffs, _, ciphertext := newTestVectors(testCtx, encryptorTest, -1, 1, t)

		verifyTestVectors(testCtx, decryptorSk0, coeffs, ciphertext, t)
	})

}

func testRelinKeyGen(testCtx *testContext, t *testing.T) {

	evaluator := testCtx.evaluator
	encryptorPk0 := testCtx.encryptorPk0
	decryptorSk0 := testCtx.decryptorSk0
	sk0Shards := testCtx.sk0Shards

	t.Run(testString("RelinKeyGen/", parties, testCtx.params), func(t *testing.T) {

		type Party struct {
			*RKGProtocol
			u      *ring.Poly
			s      *ring.Poly
			share1 RKGShare
			share2 RKGShare
		}

		rkgParties := make([]*Party, parties)

		for i := range rkgParties {
			p := new(Party)
			p.RKGProtocol = NewEkgProtocol(testCtx.params)
			p.u = p.NewEphemeralKey()
			p.s = sk0Shards[i].Get()
			p.share1, p.share2 = p.AllocateShares()
			rkgParties[i] = p
		}

		P0 := rkgParties[0]

		crpGenerator := ring.NewUniformSampler(testCtx.prng, testCtx.drckksContext.ringQP)
		crp := make([]*ring.Poly, testCtx.params.Beta())

		for i := uint64(0); i < testCtx.params.Beta(); i++ {
			crp[i] = crpGenerator.ReadNew()
		}

		// ROUND 1
		for i, p := range rkgParties {
			p.GenShareRoundOne(p.u, p.s, crp, p.share1)
			if i > 0 {
				P0.AggregateShareRoundOne(p.share1, P0.share1, P0.share1)
			}
		}

		//ROUND 2
		for i, p := range rkgParties {
			p.GenShareRoundTwo(P0.share1, p.u, p.s, crp, p.share2)
			if i > 0 {
				P0.AggregateShareRoundTwo(p.share2, P0.share2, P0.share2)
			}
		}

		evk := rckks.NewRelinKey(testCtx.params)
		P0.GenRelinearizationKey(P0.share1, P0.share2, evk)

		coeffs, _, ciphertext := newTestVectors(testCtx, encryptorPk0, -1, 1, t)

		for i := range coeffs {
			coeffs[i] *= coeffs[i]
		}

		evaluator.MulRelin(ciphertext, ciphertext, evk, ciphertext)

		evaluator.Rescale(ciphertext, testCtx.params.Scale(), ciphertext)

		require.Equal(t, ciphertext.Degree(), uint64(1))

		verifyTestVectors(testCtx, decryptorSk0, coeffs, ciphertext, t)

	})

}

func testRotKeyGenCols(testCtx *testContext, t *testing.T) {

	ringQP := testCtx.drckksContext.ringQP
	evaluator := testCtx.evaluator
	encryptorPk0 := testCtx.encryptorPk0
	decryptorSk0 := testCtx.decryptorSk0
	sk0Shards := testCtx.sk0Shards

	t.Run(testString("RotKeyGenCols/", parties, testCtx.params), func(t *testing.T) {

		type Party struct {
			*RTGProtocol
			s     *ring.Poly
			share RTGShare
		}

		pcksParties := make([]*Party, parties)
		for i := uint64(0); i < parties; i++ {
			p := new(Party)
			p.RTGProtocol = NewRotKGProtocol(testCtx.params)
			p.s = sk0Shards[i].Get()
			p.share = p.AllocateShare()
			pcksParties[i] = p
		}

		P0 := pcksParties[0]

		crpGenerator := ring.NewUniformSampler(testCtx.prng, ringQP)
		crp := make([]*ring.Poly, testCtx.params.Beta())

		for i := uint64(0); i < testCtx.params.Beta(); i++ {
			crp[i] = crpGenerator.ReadNew()
		}

		mask := ringQP.N - 1

		coeffs, _, ciphertext := newTestVectors(testCtx, encryptorPk0, -1, 1, t)

		receiver := rckks.NewCiphertext(testCtx.params, ciphertext.Degree(), ciphertext.Level(), ciphertext.Scale())

		for k := uint64(1); k < ringQP.N>>1; k <<= 1 {

			for i, p := range pcksParties {
				p.GenShare(rckks.RotationLeft, k, p.s, crp, &p.share)
				if i > 0 {
					P0.Aggregate(p.share, P0.share, P0.share)
				}
			}

			rotkey := rckks.NewRotationKeys()
			P0.Finalize(testCtx.params, P0.share, crp, rotkey)

			evaluator.RotateColumns(ciphertext, k, rotkey, receiver)

			coeffsWant := make([]float64, ringQP.N)

			for i := uint64(0); i < ringQP.N; i++ {
				coeffsWant[i] = coeffs[(i+k)&mask]
			}

			verifyTestVectors(testCtx, decryptorSk0, coeffsWant, receiver, t)
		}
	})
}


func newTestVectors(testCtx *testContext,  encryptor rckks.Encryptor, a, b float64, t *testing.T) (values []float64, plaintext *rckks.Plaintext, ciphertext *rckks.Ciphertext) {

	slots := testCtx.params.Slots()

	values = make([]float64, slots)

	for i := uint64(0); i < slots; i++ {
		values[i] = randomFloat(a, b)
	}

	values[0] = 0.607538

	plaintext = rckks.NewPlaintext(testCtx.params, testCtx.params.MaxLevel(), testCtx.params.Scale())

	testCtx.encoder.EncodeNTT(plaintext, values, slots)

	if encryptor != nil {
		ciphertext = encryptor.EncryptNew(plaintext)
	}

	return values, plaintext, ciphertext
}

func verifyTestVectors(testCtx *testContext, decryptor rckks.Decryptor, valuesWant []float64, element interface{}, t *testing.T) {

	precStats := rckks.GetPrecisionStats(testCtx.params, testCtx.encoder, decryptor, valuesWant, element)

	if *printPrecisionStats {
		t.Log(precStats.String())
	}

	require.GreaterOrEqual(t, precStats.MeanPrecision, minPrec)
}