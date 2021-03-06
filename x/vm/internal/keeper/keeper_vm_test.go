// +build unit

package keeper

import (
	"bytes"
	"encoding/binary"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/dfinance/dvm-proto/go/vm_grpc"

	"github.com/dfinance/dnode/x/common_vm"
	"github.com/dfinance/dnode/x/vm/internal/types"
)

// TODO: change listener logic to don't close it here?

// Check storage set value functional.
func TestVMKeeper_SetValue(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	ap := &vm_grpc.VMAccessPath{
		Address: input.addressBytes,
		Path:    input.pathBytes,
	}

	input.vk.setValue(input.ctx, ap, input.valueBytes)
	value := input.vk.getValue(input.ctx, ap)

	require.True(t, bytes.Equal(input.valueBytes, value))

	isExists := input.vk.hasValue(input.ctx, ap)
	require.True(t, isExists)
}

// Check get value from storage functional.
func TestVMKeeper_GetValue(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	ap := randomPath()
	input.vk.setValue(input.ctx, ap, input.valueBytes)

	value := input.vk.getValue(input.ctx, ap)
	require.Equal(t, input.valueBytes, value)

	notExistsPath := randomPath()

	var nilBytes []byte
	value = input.vk.getValue(input.ctx, notExistsPath)
	require.EqualValues(t, nilBytes, value)
}

// Check has value functional.
func TestVMKeeper_HasValue(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	ap := randomPath()

	input.vk.setValue(input.ctx, ap, input.valueBytes)

	isExists := input.vk.hasValue(input.ctx, ap)
	require.True(t, isExists)

	wrongAp := randomPath()
	isExists = input.vk.hasValue(input.ctx, wrongAp)
	require.False(t, isExists)
}

// Check deletion of key in storage.
func TestVMKeeper_DelValue(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	var emptyBytes []byte

	ap := randomPath()
	input.vk.setValue(input.ctx, ap, input.valueBytes)

	value := input.vk.getValue(input.ctx, ap)
	require.EqualValues(t, input.valueBytes, value)

	isExists := input.vk.hasValue(input.ctx, ap)
	require.True(t, isExists)

	input.vk.delValue(input.ctx, ap)

	value = input.vk.getValue(input.ctx, ap)
	require.EqualValues(t, emptyBytes, value)

	isExists = input.vk.hasValue(input.ctx, ap)
	require.False(t, isExists)
}

// Check process execution (response from VM) functional.
func TestVMKeeper_ProcessExecution(t *testing.T) {
	t.Parallel()

	// ignoring gas for now.
	input := newTestInput(true)
	defer input.Stop()

	resp := &vm_grpc.VMExecuteResponse{
		Status: &vm_grpc.VMStatus{
			Error: &vm_grpc.VMStatus_ExecutionFailure{
				ExecutionFailure: &vm_grpc.Failure{
					StatusCode: 100,
				},
			},
			Message: &vm_grpc.Message{
				Text: "this is another errorr!!!1111",
			},
		},
	}

	input.vk.processExecution(input.ctx, resp)

	procEvents := input.ctx.EventManager().Events()
	expectedEvents := types.NewContractEvents(resp)
	require.Len(t, procEvents, 2)
	require.Len(t, expectedEvents, 1)
	require.Equal(t, procEvents[1], expectedEvents[0])

	// discard without status
	resp = &vm_grpc.VMExecuteResponse{
		Status: &vm_grpc.VMStatus{
			Error: &vm_grpc.VMStatus_ExecutionFailure{
				ExecutionFailure: &vm_grpc.Failure{
					StatusCode: 100,
				},
			},
		},
	}

	ctx := input.ctx.WithEventManager(sdk.NewEventManager())
	input.vk.processExecution(ctx, resp)

	procEvents = ctx.EventManager().Events()
	expectedEvents = types.NewContractEvents(&vm_grpc.VMExecuteResponse{
		Status: &vm_grpc.VMStatus{
			Error: &vm_grpc.VMStatus_ExecutionFailure{
				ExecutionFailure: &vm_grpc.Failure{
					StatusCode: 100,
				},
			},
		},
	})
	require.Len(t, procEvents, 2)
	require.Len(t, expectedEvents, 1)
	require.Equal(t, procEvents[1], expectedEvents[0])

	// status keep
	resp = &vm_grpc.VMExecuteResponse{
		Status: &vm_grpc.VMStatus{},
	}

	ctx = input.ctx.WithEventManager(sdk.NewEventManager())
	input.vk.processExecution(ctx, resp)

	procEvents = ctx.EventManager().Events()
	expectedEvents = types.NewContractEvents(&vm_grpc.VMExecuteResponse{
		Status: &vm_grpc.VMStatus{},
	})
	require.Len(t, procEvents, 2)
	require.Len(t, expectedEvents, 1)
	require.Equal(t, procEvents[1], expectedEvents[0])

	// write set & events
	var u64Value uint64 = 100
	u64Bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(u64Bytes, u64Value)

	respEvents := make([]*vm_grpc.VMEvent, 2)
	respEvents[0] = &vm_grpc.VMEvent{
		SenderAddress: common_vm.StdLibAddress,
		EventType: &vm_grpc.LcsTag{
			TypeTag: vm_grpc.LcsType_LcsVector,
			VectorType: &vm_grpc.LcsTag{
				TypeTag: vm_grpc.LcsType_LcsU8,
			},
		},
		EventData: randomValue(32),
	}
	respEvents[1] = &vm_grpc.VMEvent{
		SenderAddress: common_vm.StdLibAddress,
		EventType: &vm_grpc.LcsTag{
			TypeTag: vm_grpc.LcsType_LcsU64,
		},
		EventData: u64Bytes,
	}

	dnEvents := make(sdk.Events, 2)
	dnEvents[0] = types.NewMoveEvent(sdk.NewInfiniteGasMeter(), respEvents[0])
	dnEvents[1] = types.NewMoveEvent(sdk.NewInfiniteGasMeter(), respEvents[1])

	writeSet := make([]*vm_grpc.VMValue, 2)
	writeSet[0] = &vm_grpc.VMValue{
		Type:  vm_grpc.VmWriteOp_Value,
		Value: randomValue(32),
		Path:  randomPath(),
	}
	writeSet[1] = &vm_grpc.VMValue{
		Type:  vm_grpc.VmWriteOp_Value,
		Value: randomValue(16),
		Path:  randomPath(),
	}

	resp = &vm_grpc.VMExecuteResponse{
		WriteSet: writeSet,
		Events:   respEvents,
		Status:   &vm_grpc.VMStatus{},
	}

	ctx = input.ctx.WithEventManager(sdk.NewEventManager())
	input.vk.processExecution(ctx, resp)
	procEvents = ctx.EventManager().Events()

	// check that everything fine with write set
	for _, write := range writeSet {
		require.True(t, input.vk.hasValue(input.ctx, write.Path))
		require.Equal(t, write.Value, input.vk.getValue(input.ctx, write.Path))
	}

	require.Len(t, procEvents, len(dnEvents)+2)

	for i, event := range procEvents[2:] {
		require.Equal(t, dnEvents[i].Type, event.Type)

		for j, attr := range event.Attributes {
			require.EqualValues(t, dnEvents[i].Attributes[j].Key, attr.Key)
			require.EqualValues(t, dnEvents[i].Attributes[j].Value, attr.Value)
		}
	}

	// check deletion
	writeSet[1] = &vm_grpc.VMValue{
		Type: vm_grpc.VmWriteOp_Deletion,
		Path: writeSet[1].Path,
	}

	resp = &vm_grpc.VMExecuteResponse{
		WriteSet: writeSet,
		Status:   &vm_grpc.VMStatus{},
	}

	ctx = input.ctx.WithEventManager(sdk.NewEventManager())
	input.vk.processExecution(ctx, resp)
	procEvents = ctx.EventManager().Events()

	require.Len(t, procEvents, 2)

	require.False(t, input.vk.hasValue(input.ctx, writeSet[1].Path))
	require.Nil(t, input.vk.getValue(input.ctx, writeSet[1].Path))
}

// Check returned write set procession.
func TestVMKeeper_ProcessWriteSet(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	writeSet := make([]*vm_grpc.VMValue, 2)
	writeSet[0] = &vm_grpc.VMValue{
		Type:  vm_grpc.VmWriteOp_Value,
		Value: randomValue(32),
		Path:  randomPath(),
	}
	writeSet[1] = &vm_grpc.VMValue{
		Type:  vm_grpc.VmWriteOp_Value,
		Value: randomValue(16),
		Path:  randomPath(),
	}

	input.vk.processWriteSet(input.ctx, writeSet)

	// now read storage and check results
	values := make([][]byte, 2)
	values[0] = input.vk.getValue(input.ctx, writeSet[0].Path)
	values[1] = input.vk.getValue(input.ctx, writeSet[1].Path)

	for i, write := range writeSet {
		require.True(t, input.vk.hasValue(input.ctx, write.Path))
		require.Equal(t, write.Value, values[i])
	}

	// check delete op
	delSet := make([]*vm_grpc.VMValue, 2)
	delSet[0] = &vm_grpc.VMValue{
		Type: vm_grpc.VmWriteOp_Deletion,
		Path: writeSet[0].Path,
	}
	delSet[1] = &vm_grpc.VMValue{
		Type: vm_grpc.VmWriteOp_Deletion,
		Path: writeSet[1].Path,
	}

	input.vk.processWriteSet(input.ctx, delSet)

	for _, del := range delSet {
		require.False(t, input.vk.hasValue(input.ctx, del.Path))
		value := input.vk.getValue(input.ctx, del.Path)
		require.Nil(t, value)
	}
}

// Status keep (4001) still doesn't contains error.
func TestVMKeeper_ExecStatusKeeperNotAnError(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	resp := &vm_grpc.VMExecuteResponse{
		WriteSet: nil,
		Events:   nil,
		Status:   &vm_grpc.VMStatus{},
	}

	input.vk.processExecution(input.ctx, resp)
	events := input.ctx.EventManager().Events()

	require.EqualValues(t, sdk.EventTypeMessage, events[0].Type)
	require.EqualValues(t, sdk.AttributeKeyModule, events[0].Attributes[0].Key)
	require.EqualValues(t, types.ModuleName, events[0].Attributes[0].Value)

	require.EqualValues(t, types.EventTypeContractStatus, events[1].Type)
	require.EqualValues(t, types.AttributeStatus, events[1].Attributes[0].Key)
	require.EqualValues(t, types.AttributeValueStatusKeep, events[1].Attributes[0].Value)

	for _, attr := range events[0].Attributes {
		require.NotEqual(t, []byte(types.AttributeValueStatusError), attr.Key)
	}
}

// When status still keep but returns error (and it could exists).
func TestVMKeeper_ExecKeepAndError(t *testing.T) {
	t.Parallel()

	input := newTestInput(true)
	defer input.Stop()

	errMessage := "aborted error!11111!1!!!"

	resp := &vm_grpc.VMExecuteResponse{
		WriteSet: nil,
		Events:   nil,
		Status: &vm_grpc.VMStatus{
			Error: &vm_grpc.VMStatus_MoveError{
				MoveError: &vm_grpc.MoveError{
					StatusCode: 3002,
				},
			},
			Message: &vm_grpc.Message{
				Text: errMessage,
			},
		},
	}

	input.vk.processExecution(input.ctx, resp)
	events := input.ctx.EventManager().Events()

	require.EqualValues(t, sdk.EventTypeMessage, events[0].Type)
	require.EqualValues(t, sdk.AttributeKeyModule, events[0].Attributes[0].Key)
	require.EqualValues(t, types.ModuleName, events[0].Attributes[0].Value)

	require.EqualValues(t, types.EventTypeContractStatus, events[1].Type)
	require.EqualValues(t, types.AttributeStatus, events[1].Attributes[0].Key)
	require.EqualValues(t, types.AttributeValueStatusDiscard, events[1].Attributes[0].Value)

	require.EqualValues(t, types.AttributeErrMajorStatus, events[1].Attributes[1].Key)
	majorCode := resp.Status.Error.(*vm_grpc.VMStatus_MoveError).MoveError.StatusCode
	require.EqualValues(t, strconv.FormatUint(majorCode, 10), events[1].Attributes[1].Value)

	require.EqualValues(t, types.AttributeErrSubStatus, events[1].Attributes[2].Key)
	require.EqualValues(t, "0", events[1].Attributes[2].Value)

	require.EqualValues(t, types.AttributeErrMessage, events[1].Attributes[3].Key)
	require.EqualValues(t, errMessage, events[1].Attributes[3].Value)
}
