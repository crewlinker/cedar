package cedar_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tetratelabs/wazero"

	_ "embed"
)

//go:embed cedarwasm/cedar.wasm
var wasm []byte

func TestCedar(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "cedar")
}

var _ = Describe("Cedar", func() {
	It("should run add from wasm", func(ctx context.Context) {
		runt := wazero.NewRuntime(ctx)
		DeferCleanup(runt.Close)

		mod, err := runt.Instantiate(ctx, wasm)
		Expect(err).ToNot(HaveOccurred())

		Expect(mod.ExportedFunctionDefinitions()).To(HaveKey("add"))

		res, err := mod.ExportedFunction("add").Call(ctx, 100, 10)
		Expect(err).ToNot(HaveOccurred())

		Expect(res[0]).To(BeNumerically("==", 110))
	})
})
