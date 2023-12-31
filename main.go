package main

import (
	"dynamicfees/db/postgresql"
	"dynamicfees/mempoolspace"
	"dynamicfees/whatthefee"
	"encoding/json"

	"log"
	"os"

	starlarkjson "go.starlark.net/lib/json"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

func starlarkMakeWFees(w whatthefee.FeerateEstimation) *starlark.Dict {
	wf := new(starlark.Dict)

	ii := []starlark.Value{}
	for _, i := range w.Index {
		ii = append(ii, starlark.MakeInt64(i))
	}
	wf.SetKey(starlark.String("index"), starlark.NewList(ii))

	cc := []starlark.Value{}
	for _, c := range w.Columns {
		cc = append(cc, starlark.String(c))
	}
	wf.SetKey(starlark.String("columns"), starlark.NewList(cc))

	dd := []starlark.Value{}
	for _, line := range w.Data {
		ll := []starlark.Value{}
		for _, level := range line {
			ll = append(ll, starlark.MakeInt64(level))
		}
		dd = append(dd, starlark.NewList(ll))
	}
	wf.SetKey(starlark.String("data"), starlark.NewList(dd))

	return wf
}

func main() {
	databaseUrl := os.Getenv("DATABASE_URL")
	pool, err := postgresql.PgConnect(databaseUrl)
	if err != nil {
		log.Fatalf("pgConnect() error: %v", err)
	}
	store := postgresql.NewPostgresStore(pool)

	mFees, err := mempoolspace.GetRecommendedFees(os.Getenv("MEMPOOL_API_BASE_URL"))
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	//fmt.Printf("%#v\n", *mFees)
	wFees, err := whatthefee.GetFeerateEstimation(os.Getenv("WHATTHEFEE_API_BASE_URL"))
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	//fmt.Printf("%#v\n", *wFees)
	allParams, err := store.GetAllParams()
	if err != nil {
		log.Printf("Error: %v", err)
	}
	for token, p := range allParams {
		//fmt.Printf("token: %v\nStarlark: %v\nExtraData: %s\n", token, p.Starlark, p.ExtraData)
		if p.Starlark != "" {
			for validity, op := range p.OpeningParams {
				//fmt.Printf("  validity:%v->%#v\n", validity, op)

				opDict := new(starlark.Dict)
				opDict.SetKey(starlark.String("min_msat"), starlark.MakeUint64(op.MinFeeMsat))
				opDict.SetKey(starlark.String("proportional"), starlark.MakeUint64(uint64(op.Proportional)))
				opDict.SetKey(starlark.String("max_idle_time"), starlark.MakeUint64(uint64(op.MaxIdleTime)))
				opDict.SetKey(starlark.String("max_client_to_self_delay"), starlark.MakeUint64(uint64(op.MaxClientToSelfDelay)))
				opDict.SetKey(starlark.String("extra_data"), starlark.String(op.ExtraData))

				mf := new(starlark.Dict)
				mf.SetKey(starlark.String("fastestFee"), starlark.MakeUint(mFees.FastestFee))
				mf.SetKey(starlark.String("halfHourFee"), starlark.MakeUint(mFees.FastestFee))
				mf.SetKey(starlark.String("hourFee"), starlark.MakeUint(mFees.FastestFee))
				mf.SetKey(starlark.String("economyFee"), starlark.MakeUint(mFees.FastestFee))
				mf.SetKey(starlark.String("minimumFee"), starlark.MakeUint(mFees.FastestFee))

				globals := starlark.StringDict{
					"token":             starlark.String(token),
					"validity":          starlark.MakeInt64(validity),
					"token_extra_data":  starlark.String(p.ExtraData),
					"opening_params":    opDict,
					"mempoolspace_fees": mf,
					"whatthefee_fees":   starlarkMakeWFees(*wFees),
				}
				for _, k := range starlarkjson.Module.Members.Keys() {
					globals[k] = starlarkjson.Module.Members[k]
				}
				g, err := starlark.ExecFileOptions(&syntax.FileOptions{}, &starlark.Thread{Name: "dynamicFees"}, "dynamicFees.star", p.Starlark, globals)
				if err != nil {
					log.Printf("Error in starlark.ExecFileOptions: %v\n%v", err, p.Starlark)
					continue
				}

				newOp := postgresql.OpeningParams{}
				sNewOp, ok := g["new_opening_params"].(*starlark.Dict)
				if ok {
					v, found, err := sNewOp.Get(starlark.String("min_msat"))
					if err == nil && found {
						var minSat uint64
						err = starlark.AsInt(v, &minSat)
						if err == nil {
							newOp.MinFeeMsat = minSat
						}
					}
					v, found, err = sNewOp.Get(starlark.String("proportional"))
					if err == nil && found {
						var proportional uint32
						err = starlark.AsInt(v, &proportional)
						if err == nil {
							newOp.Proportional = proportional
						}
					}
					v, found, err = sNewOp.Get(starlark.String("extra_data"))
					if err == nil && found {
						sed, ok := v.(starlark.String)
						if ok {
							newOp.ExtraData = json.RawMessage(sed.GoString())
						}
					}
				}
				//jop, err := json.Marshal(newOp)
				//fmt.Printf("newOp - json: %s, err: %v\n", jop, err)
				store.SetOpeningParams(token, validity, newOp)
			}
		}
	}
}
