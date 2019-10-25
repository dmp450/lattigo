package dckks

import (
	"fmt"
	"log"
	"testing"

	"github.com/ldsec/lattigo/ckks"
	"github.com/ldsec/lattigo/ring"
)

type benchParams struct {
	parties       uint64
	params        *ckks.Parameters
	sigmaSmudging float64
}

type benchContext struct {
	ckkscontext *ckks.CkksContext
	sk0         *ckks.SecretKey
	sk1         *ckks.SecretKey
	pk0         *ckks.PublicKey
	pk1         *ckks.PublicKey
	cprng       *CRPGenerator
}

func Benchmark_DCKKSScheme(b *testing.B) {

	var err error

	params := []benchParams{

		{parties: 5, params: ckks.DefaultParams[14], sigmaSmudging: 6.4},
	}

	for _, param := range params {

		benchcontext := new(benchContext)

		if benchcontext.ckkscontext, err = ckks.NewCkksContext(param.params); err != nil {
			log.Fatal(err)
		}

		log.Printf("Benchmarks for parties=%d/logN=%d/logQ=%d/levels=%d/sigma=%.2f/sigmaSmudging=%.2f",
			param.parties,
			benchcontext.ckkscontext.LogN(),
			benchcontext.ckkscontext.LogQ(),
			benchcontext.ckkscontext.Levels(),
			benchcontext.ckkscontext.Sigma(),
			param.sigmaSmudging)

		kgen := benchcontext.ckkscontext.NewKeyGenerator()

		benchcontext.sk0, benchcontext.pk0 = kgen.NewKeyPair()
		benchcontext.sk1, benchcontext.pk1 = kgen.NewKeyPair()

		benchcontext.cprng, err = NewCRPGenerator(nil, benchcontext.ckkscontext.ContextKeys())
		if err != nil {
			log.Fatal(err)
		}

		benchcontext.cprng.Seed([]byte{})

		//bench_EKG(param, benchcontext, b)
		bench_EKGNaive(param, benchcontext, b)
		bench_CKG(param, benchcontext, b)
		bench_CKS(param, benchcontext, b)
		bench_PCKS(param, benchcontext, b)
		bench_BOOT(param, benchcontext, b)

	}
}
/*
func bench_EKG(params benchParams, context *benchContext, b *testing.B) {
	// EKG

	EkgProtocol := NewEkgProtocol(context.ckkscontext.ContextKeys(), context.ckkscontext.KeySwitchPrimes())

	crp := make([]*ring.Poly, context.ckkscontext.Levels()+1)

	for i := uint64(0); i < context.ckkscontext.Levels()+1; i++ {
		crp[i] = context.cprng.Clock()
	}

	samples := make([][]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		samples[i] = make([]*ring.Poly, context.ckkscontext.Levels())
		samples[i] = EkgProtocol.GenSamples(context.sk0.Get(), context.sk1.Get(), crp)
	}

	aggregatedSamples := make([][][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		aggregatedSamples[i] = EkgProtocol.Aggregate(context.sk1.Get(), samples, crp)
	}

	keySwitched := make([][]*ring.Poly, params.parties)

	sum := EkgProtocol.Sum(aggregatedSamples)
	for i := uint64(0); i < params.parties; i++ {
		keySwitched[i] = EkgProtocol.KeySwitch(context.sk0.Get(), context.sk1.Get(), sum)
	}

	//EKG_V2_Round_0
	b.Run(fmt.Sprintf("EKG_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.GenSamples(context.sk0.Get(), context.sk1.Get(), crp)
		}
	})

	//EKG_V2_Round_1
	b.Run(fmt.Sprintf("EKG_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.Aggregate(context.sk1.Get(), samples, crp)
		}
	})

	//EKG_V2_Round_2
	b.Run(fmt.Sprintf("EKG_Round2"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.KeySwitch(context.sk1.Get(), context.sk1.Get(), EkgProtocol.Sum(aggregatedSamples))
		}
	})

	//EKG_V2_Round_3
	b.Run(fmt.Sprintf("EKG_Round3"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			EkgProtocol.ComputeEVK(keySwitched, sum)
		}
	})
}
*/
func bench_EKGNaive(params benchParams, context *benchContext, b *testing.B) {
	// EKG_Naive
	ekgV2Naive := NewEkgProtocolNaive(context.ckkscontext.ContextKeys(), context.ckkscontext.KeySwitchPrimes())

	// [nParties][CrtDecomp][WDecomp][2]
	samples := make([][][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		samples[i] = ekgV2Naive.GenSamples(context.sk0.Get(), context.pk0.Get())
	}

	aggregatedSamples := make([][][2]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		aggregatedSamples[i] = ekgV2Naive.Aggregate(context.sk0.Get(), context.pk0.Get(), samples)
	}

	//EKG_V2_Naive_Round_0
	b.Run(fmt.Sprintf("EKG_Naive_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ekgV2Naive.GenSamples(context.sk0.Get(), context.pk1.Get())
		}
	})

	//EKG_V2_Naive_Round_1
	b.Run(fmt.Sprintf("EKG_Naive_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ekgV2Naive.Aggregate(context.sk0.Get(), context.pk1.Get(), samples)
		}
	})

	//EKG_V2_Naive_Round_2
	b.Run(fmt.Sprintf("EKG_Naive_Round2"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ekgV2Naive.Finalize(aggregatedSamples)
		}
	})
}

func bench_CKG(params benchParams, context *benchContext, b *testing.B) {

	//CKG
	ckgInstance := NewCKG(context.ckkscontext.ContextKeys(), context.cprng.Clock())
	ckgInstance.GenShare(context.sk0.Get())

	shares := make([]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		shares[i] = ckgInstance.GetShare()
	}

	// CKG_Round_0
	b.Run(fmt.Sprintf("CKG_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ckgInstance.GenShare(context.sk0.Get())
		}
	})

	// CKG_Round_1
	b.Run(fmt.Sprintf("CKG_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ckgInstance.AggregateShares(shares)
			//ckgInstance.Finalize()
		}
	})
}

func bench_CKS(params benchParams, context *benchContext, b *testing.B) {
	//CKS

	cksInstance := NewCKS(context.sk0.Get(), context.sk1.Get(), context.ckkscontext, params.sigmaSmudging)

	ciphertext := context.ckkscontext.NewRandomCiphertext(1, context.ckkscontext.Levels()-1, context.ckkscontext.Scale())

	hi := make([]*ring.Poly, params.parties)
	for i := uint64(0); i < params.parties; i++ {
		hi[i] = cksInstance.KeySwitch(ciphertext.Value()[1])
	}

	// CKS_Round_0
	b.Run(fmt.Sprintf("CKS_Round0"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cksInstance.KeySwitch(ciphertext.Value()[1])
		}
	})

	// CKS_Round_1
	b.Run(fmt.Sprintf("CKS_Round1"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cksInstance.Aggregate(ciphertext.Value()[0], hi)
		}
	})
}

func bench_PCKS(params benchParams, context *benchContext, b *testing.B) {
	// TODO
}

func bench_BOOT(params benchParams, context *benchContext, b *testing.B) {

	crp := context.ckkscontext.ContextQ().NewPoly()
	ciphertext := context.ckkscontext.NewRandomCiphertext(1, 2, context.ckkscontext.Scale())

	refreshShares := make([]*RefreshShares, params.parties)

	b.Run(fmt.Sprintf("BOOT_GenShares"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			refreshShares[0] = GenRefreshShares(context.sk0, 2, params.parties, context.ckkscontext, ciphertext.Value()[0], crp)
		}
	})

	for i := uint64(1); i < params.parties; i++ {
		refreshShares[i] = GenRefreshShares(context.sk0, 2, params.parties, context.ckkscontext, ciphertext.Value()[0], crp)
	}

	b.Run(fmt.Sprintf("BOOT_Recrypt"), func(b *testing.B) {
		for i := 0; i < b.N; i++ {

			Refresh(ciphertext, refreshShares, context.ckkscontext, crp)

			b.StopTimer()
			ciphertext.Value()[0].Coeffs = ciphertext.Value()[0].Coeffs[:2]
			b.StartTimer()
		}
	})
}
