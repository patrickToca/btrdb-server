package qtree

import (
	"testing"
	"math/rand"
	"log"
	"fmt"
	"time"
	bstore "cal-sdb.org/quasar/bstoreEmu"
)

func init() {
	sd := time.Now().Unix()
	fmt.Printf(">>>> USING %v AS SEED <<<<<", sd)
	rand.Seed(sd)
}
func GenBrk(avg uint64, spread uint64) chan uint64{
	rv := make(chan uint64)
	go func() {
		for {
			num := int64(avg)
			num -= int64(spread/2)
			num += rand.Int63n(int64(spread))
			rv <- uint64(num)
		}
	} ()
	return rv
}

func TestQT2_PW(t *testing.T){
	te := int64(4096)
	tdat := GenData(0, 4096, 1, 0, func(_ int64) float64 {return rand.Float64()})
	if int64(len(tdat)) != te {
		log.Panic("GenDat messed up a bit")
	}
	tr, uuid := MakeWTree()
	tr.InsertValues(tdat)
	tr.Commit()
	var err error
	tr, err = NewReadQTree(_bs, uuid, bstore.LatestGeneration)
	if err != nil {
		t.Error(err)
	}
	
	moddat := make([]StatRecord, len(tdat))
	for i,v := range tdat {
		moddat[i] = StatRecord {
			Time:v.Time,
			Count:1,
			Min:v.Val,
			Mean:v.Val,
			Max:v.Val,
		}
	}
	for pwi:=uint8(0); pwi<12;pwi++ {
		qrydat, err := tr.QueryStatisticalValuesBlock(0, te, pwi)
		if err != nil {
			log.Panic(err)
		}
		if int64(len(qrydat)) != te >> pwi {
			t.Log("len of qrydat mismatch %v vs %v", len(qrydat), te>>pwi)
			log.Printf("qry dat %+v", qrydat)
			t.FailNow()
		} else {
			t.Log("LEN MATCH %v",len(qrydat))
		}
		min := func (a float64, b float64) float64{
			if a<b {return a}
			return b
		}
		max := func (a float64, b float64) float64{
			if a>b {return a}
			return b
		}
		moddat2 := make([]StatRecord, len(moddat)/2)
		for i:=0; i < len(moddat)/2; i++ {
			nmean := moddat[2*i].Mean*float64(moddat[2*i].Count) +
					 moddat[2*i+1].Mean*float64(moddat[2*i+1].Count)
			nmean /= float64(moddat[2*i].Count + moddat[2*i+1].Count)
			
			moddat2[i] = StatRecord {
				Time:moddat[2*i].Time,
				Count:moddat[2*i].Count + moddat[2*i+1].Count,
				Min: min(moddat[2*i].Min, moddat[2*i+1].Min),
				Mean: nmean,
			    Max: max(moddat[2*i].Max, moddat[2*i+1].Max),
			}
		}
	}
	
}
func TestQT2_A(t *testing.T){
	gs := int64(20+rand.Intn(10))*365*DAY
	ge := int64(30+rand.Intn(10))*365*DAY
	freq := uint64(rand.Intn(10))*HOUR
	varn := uint64(30*MINUTE)+1
	tdat := GenData(gs,ge, freq, varn, 
		func(_ int64) float64 {return rand.Float64()})
	log.Printf("generated %v records",len(tdat))
	tr, uuid := MakeWTree()
	log.Printf("geneated tree %v",tr.gen.Uuid().String())
	tr.Commit()
	
	idx := uint64(0)
	brks := GenBrk(100,50)
	loops := GenBrk(4,4)
	for ;idx<uint64(len(tdat)); {
		tr := LoadWTree(uuid)
		loop := <- loops
		for i:= uint64(0); i<loop; i++ {
			brk := <- brks
			if idx+brk >= uint64(len(tdat)) {
				brk = uint64(len(tdat)) - idx
			}
			if brk == 0 {
				continue
			}
			tr.InsertValues(tdat[idx:idx+brk])
			idx += brk
		}
		tr.Commit()
	}
	
	rtr, err := NewReadQTree(_bs, uuid, bstore.LatestGeneration) 
	if err != nil {
		log.Panic(err)
	}
	rval, err := rtr.ReadStandardValuesBlock(gs, ge+int64(2*varn))
	if err != nil {
		log.Panic(err)
	}
	log.Printf("wrote %v, read %v", len(tdat), len(rval))
	CompareData(tdat, rval)
}

