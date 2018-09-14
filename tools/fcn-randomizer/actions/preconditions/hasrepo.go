package preconditions

import (
	"context"
	"os"
	"path/filepath"

	logging "github.com/ipfs/go-log"
	//klgwriter "github.com/ipfs/go-log/writer"
	"github.com/ipfs/iptb/testbed/interfaces"
)

var log = logging.Logger("precondition")

var ConditionName = "HasRepo"

/*
func init() {
	logging.SetAllLoggers(4)
	file, err := os.Create("../auditlogs.json")
	if err != nil {
		panic(err)
	}
	lgwriter.WriterGroup.AddWriter(file)
}
*/

type HasRepo struct {
}

func (h *HasRepo) Name() string {
	return ConditionName
}

func (h *HasRepo) Condition(ctx context.Context, n testbedi.Core) (pass bool, err error) {
	ctx = log.Start(ctx, h.Name())
	defer func() {
		log.SetTags(ctx, map[string]interface{}{
			"node": n,
			"pass": pass,
		})
		log.FinishWithErr(ctx, err)
	}()

	//checking that there is a config is probably good enough for mvp
	if _, err := os.Stat(filepath.Join(n.Dir(), "config.toml")); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
