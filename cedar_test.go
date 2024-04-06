package cedar_test

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	_ "embed"
)

//go:embed cedarwasm/cedar.wasm
var wasm []byte

func TestCedar(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "cedar")
}

func CountNumPolicies(ctx context.Context, mod api.Module) (uint64, error) {
	res, err := mod.ExportedFunction("count_num_policies").Call(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to call count_num_policies: %w", err)
	}

	return res[0], nil
}

func LoadPolicies(ctx context.Context, mod api.Module, policies string) (uint64, error) {
	allocated, err := mod.ExportedFunction("allocate").Call(ctx, uint64(len(policies)))
	if err != nil {
		return 0, fmt.Errorf("failed to allocate: %w", err)
	}

	if !mod.Memory().WriteString(uint32(allocated[0]), policies) {
		return 0, fmt.Errorf("failed to write policies to memory")
	}

	res, err := mod.ExportedFunction("load_policies").Call(ctx, allocated[0])
	if err != nil {
		return 0, fmt.Errorf("failed to load policies: %w", err)
	}

	return res[0], nil
}

var (
	emptyPolicy   = ``
	invalidPolicy = `bogus`
	policies1     = `
permit(
	principal == User::"alice", 
	action == Action::"view", 
	resource == File::"93"
);`
)

var _ = Describe("Cedar", func() {
	var runt wazero.Runtime
	var mod api.Module

	BeforeEach(func(ctx context.Context) {
		var err error

		runt = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
		DeferCleanup(runt.Close)

		mod, err = runt.Instantiate(ctx, wasm)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should error with invalid policy", func(ctx context.Context) {
		Expect(LoadPolicies(ctx, mod, invalidPolicy)).To(Equal(uint64(1002)))
	})

	It("should error when loading empty policy twice", func(ctx context.Context) {
		Expect(LoadPolicies(ctx, mod, emptyPolicy)).To(Equal(uint64(0)))
		Expect(LoadPolicies(ctx, mod, emptyPolicy)).To(Equal(uint64(1000)))
	})

	It("should load policies", func(ctx context.Context) {
		Expect(LoadPolicies(ctx, mod, policies1)).To(Equal(uint64(0)))
		Expect(CountNumPolicies(ctx, mod)).To(Equal(uint64(1)))
	})

	It("should load empty policy", func(ctx context.Context) {
		Expect(LoadPolicies(ctx, mod, emptyPolicy)).To(Equal(uint64(0)))
		Expect(CountNumPolicies(ctx, mod)).To(Equal(uint64(0)))
	})
})
