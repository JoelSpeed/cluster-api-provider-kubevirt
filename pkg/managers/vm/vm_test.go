package vm

import (
	"errors"
	"fmt"
	"testing"

	machinev1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetesclient "k8s.io/client-go/kubernetes"

	"github.com/golang/mock/gomock"
	kubevirtClient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client"
	mockkubevirtclient "github.com/kubevirt/cluster-api-provider-kubevirt/pkg/client/mock"
	"gotest.tools/assert"
	kubevirtapiv1 "kubevirt.io/client-go/api/v1"
)

func initializeMachine(t *testing.T, mockKubevirtClient *mockkubevirtclient.MockClient, labels map[string]string, providerID string) *machinev1.Machine {
	machine, stubMachineErr := stubMachine(labels, providerID)

	if stubMachineErr != nil {
		t.Fatalf("Unable to build test machine manifest: %v", stubMachineErr)
		return nil
	}

	return machine
}

func TestCreate(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr string
		wantCreateErr          string
		ClientCreateError      error
		labels                 map[string]string
		providerID             string
		wantVMToBeReady        bool
	}{
		{
			name:                   "Create a VM",
			wantValidateMachineErr: "",
			wantCreateErr:          "",
			ClientCreateError:      nil,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        true,
		},
		{
			name:                   "Create a VM not ready and fail",
			wantValidateMachineErr: "",
			wantCreateErr:          "",
			ClientCreateError:      nil,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Create a VM from unlabeled machine and fail",
			wantValidateMachineErr: fmt.Sprintf("%s: failed validating machine provider spec: %v: missing %q label", mahcineName, mahcineName, machinev1.MachineClusterIDLabel),
			wantCreateErr:          "",
			ClientCreateError:      nil,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             "",
			wantVMToBeReady:        true,
		},
		{
			name:                   "Create a VM with an error in the client-go and fail",
			wantValidateMachineErr: "",
			wantCreateErr:          "failed to create virtual machine: client error",
			ClientCreateError:      errors.New("client error"),
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)
			machine := initializeMachine(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			virtualMachine, virtualMachineErr := machineToVirtualMachine(machine, "SourceTestPvcName")
			if virtualMachineErr != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", virtualMachineErr)
			}

			returnVM, returnVMErr := machineToVirtualMachine(machine, "SourceTestPvcName")
			if returnVMErr != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", returnVMErr)
			}
			returnVM.Status.Ready = tc.wantVMToBeReady

			mockKubevirtClient.EXPECT().CreateVirtualMachine(defaultNamespace, virtualMachine).Return(returnVM, tc.ClientCreateError).AnyTimes()

			kubevirtClientMockBuilder := func(kubernetesClient *kubernetesclient.Clientset, secretName, namespace string) (kubevirtClient.Client, error) {
				return mockKubevirtClient, nil
			}
			providerVMInstance := New(kubevirtClientMockBuilder, nil)
			createVMErr := providerVMInstance.Create(machine)
			if tc.wantValidateMachineErr != "" {
				assert.Equal(t, tc.wantValidateMachineErr, createVMErr.Error())
			} else if tc.wantCreateErr != "" {
				assert.Equal(t, tc.wantCreateErr, createVMErr.Error())
			} else if !tc.wantVMToBeReady {
				assert.Equal(t, createVMErr.Error(), "requeue in: 20s")
			} else {
				assert.Equal(t, createVMErr, nil)
			}
		})
	}

}

func TestDelete(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr string
		wantGetErr             string
		wantDeleteErr          string
		clientGetError         error
		clientDeleteError      error
		emptyGetVM             bool
		labels                 map[string]string
		providerID             string
	}{
		{
			name:                   "Delete a VM successfully",
			wantValidateMachineErr: "",
			wantGetErr:             "",
			clientGetError:         nil,
			wantDeleteErr:          "",
			clientDeleteError:      nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
		},
		{
			name:                   "Delete a VM from unlabeled machine and fail",
			wantValidateMachineErr: fmt.Sprintf("%s: failed validating machine provider spec: %v: missing %q label", mahcineName, mahcineName, machinev1.MachineClusterIDLabel),
			wantGetErr:             "",
			clientGetError:         nil,
			wantDeleteErr:          "",
			clientDeleteError:      nil,
			emptyGetVM:             false,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             "",
		},
		{
			name:                   "Delete deleting VM with getting error and fail",
			wantValidateMachineErr: "",
			wantGetErr:             "client error",
			clientGetError:         errors.New("client error"),
			wantDeleteErr:          "",
			clientDeleteError:      nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
		},
		{
			name:                   "Delete a nonexistent VM and fail",
			wantValidateMachineErr: "",
			wantGetErr:             "",
			clientGetError:         nil,
			wantDeleteErr:          "",
			clientDeleteError:      nil,
			emptyGetVM:             true,
			labels:                 nil,
			providerID:             "",
		},
		{
			name:                   "Delete a VM with an error in the client-go and fail",
			wantValidateMachineErr: "",
			wantGetErr:             "",
			clientGetError:         nil,
			wantDeleteErr:          "failed to delete VM: client error",
			clientDeleteError:      errors.New("client error"),
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)

			machine := initializeMachine(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			virtualMachine, virtualMachineErr := machineToVirtualMachine(machine, "SourceTestPvcName")
			if virtualMachineErr != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", virtualMachineErr)
			}

			var returnVM *kubevirtapiv1.VirtualMachine
			if !tc.emptyGetVM {
				returnVM = virtualMachine
			}
			mockKubevirtClient.EXPECT().GetVirtualMachine(defaultNamespace, virtualMachine.Name).Return(returnVM, tc.clientGetError).AnyTimes()
			mockKubevirtClient.EXPECT().DeleteVirtualMachine(defaultNamespace, virtualMachine.Name, gomock.Any()).Return(tc.clientDeleteError).AnyTimes()

			kubevirtClientMockBuilder := func(kubernetesClient *kubernetesclient.Clientset, secretName, namespace string) (kubevirtClient.Client, error) {
				return mockKubevirtClient, nil
			}
			providerVMInstance := New(kubevirtClientMockBuilder, nil)
			deleteVMErr := providerVMInstance.Delete(machine)

			if tc.wantValidateMachineErr != "" {
				assert.Equal(t, tc.wantValidateMachineErr, deleteVMErr.Error())
			} else if tc.wantGetErr != "" {
				assert.Equal(t, tc.wantGetErr, deleteVMErr.Error())
			} else if tc.wantDeleteErr != "" {
				assert.Equal(t, tc.wantDeleteErr, deleteVMErr.Error())
			} else {
				assert.Equal(t, deleteVMErr, nil)
			}
		})
	}

}

func TestExists(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name           string
		clientGetError error
		emptyGetVM     bool
		isExist        bool
		labels         map[string]string
		providerID     string
	}{
		{
			name:           "Validate existence VM",
			clientGetError: nil,
			emptyGetVM:     false,
			isExist:        true,
			labels:         nil,
			providerID:     "",
		},
		{
			name:           "Validate non existence VM",
			clientGetError: nil,
			emptyGetVM:     true,
			isExist:        false,
			labels:         nil,
			providerID:     "",
		},
		{
			name:           "Validate an error in get VM",
			clientGetError: errors.New("client error"),
			emptyGetVM:     true,
			isExist:        false,
			labels:         nil,
			providerID:     "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)

			machine := initializeMachine(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			virtualMachine, virtualMachineErr := machineToVirtualMachine(machine, "SourceTestPvcName")
			if virtualMachineErr != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", virtualMachineErr)
			}

			var returnVM *kubevirtapiv1.VirtualMachine
			if !tc.emptyGetVM {
				returnVM = virtualMachine
			}

			mockKubevirtClient.EXPECT().GetVirtualMachine(defaultNamespace, virtualMachine.Name).Return(returnVM, tc.clientGetError).AnyTimes()

			kubevirtClientMockBuilder := func(kubernetesClient *kubernetesclient.Clientset, secretName, namespace string) (kubevirtClient.Client, error) {
				return mockKubevirtClient, nil
			}
			providerVMInstance := New(kubevirtClientMockBuilder, nil)
			existsVM, existsVMErr := providerVMInstance.Exists(machine)

			if tc.clientGetError != nil {
				assert.Equal(t, tc.clientGetError.Error(), existsVMErr.Error())
			} else if tc.emptyGetVM {
				assert.Equal(t, existsVMErr, nil)
				assert.Equal(t, existsVM, false)
			} else {
				assert.Equal(t, existsVMErr, nil)
				assert.Equal(t, existsVM, tc.isExist)
			}
		})
	}

}

func TestUpdate(t *testing.T) {
	// TODO add a case of setProviderID and setMachineCloudProviderSpecifics failure
	cases := []struct {
		name                   string
		wantValidateMachineErr string
		wantUpdateErr          string
		clientGetError         error
		clientUpdateError      error
		emptyGetVM             bool
		labels                 map[string]string
		providerID             string
		wantVMToBeReady        bool
	}{
		{
			name:                   "Update a VM",
			wantValidateMachineErr: "",
			wantUpdateErr:          "",
			clientGetError:         nil,
			clientUpdateError:      nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             fmt.Sprintf("kubevirt:///%s/%s", defaultNamespace, mahcineName),
			wantVMToBeReady:        true,
		},
		{
			name:                   "Update a VM that never be ready",
			wantValidateMachineErr: "",
			wantUpdateErr:          "",
			clientGetError:         nil,
			clientUpdateError:      nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Update a VM from unlabeled machine and fail",
			wantValidateMachineErr: fmt.Sprintf("%s: failed validating machine provider spec: %v: missing %q label", mahcineName, mahcineName, machinev1.MachineClusterIDLabel),
			wantUpdateErr:          "",
			clientGetError:         nil,
			clientUpdateError:      nil,
			emptyGetVM:             false,
			labels:                 map[string]string{machinev1.MachineClusterIDLabel: ""},
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Update a VM with getting error and fail",
			wantValidateMachineErr: "",
			wantUpdateErr:          "",
			clientGetError:         errors.New("client error"),
			clientUpdateError:      nil,
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Update a nonexistent VM and fail",
			wantValidateMachineErr: "",
			wantUpdateErr:          "",
			clientGetError:         nil,
			clientUpdateError:      nil,
			emptyGetVM:             true,
			labels:                 nil,
			providerID:             "",
			wantVMToBeReady:        false,
		},
		{
			name:                   "Delete a VM with an error in the client-go and fail",
			wantValidateMachineErr: "",
			clientGetError:         nil,
			clientUpdateError:      errors.New("client error"),
			wantUpdateErr:          "failed to update VM: client error",
			emptyGetVM:             false,
			labels:                 nil,
			providerID:             "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockKubevirtClient := mockkubevirtclient.NewMockClient(mockCtrl)

			machine := initializeMachine(t, mockKubevirtClient, tc.labels, tc.providerID)
			if machine == nil {
				t.Fatalf("Unable to create the stub machine object")
			}

			virtualMachine, virtualMachineErr := machineToVirtualMachine(machine, "SourceTestPvcName")
			if virtualMachineErr != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", virtualMachineErr)
			}

			var getReturnVM *kubevirtapiv1.VirtualMachine
			if !tc.emptyGetVM {
				returnVMResult, returnVMErr := machineToVirtualMachine(machine, "SourceTestPvcName")
				if returnVMErr != nil {
					t.Fatalf("Unable to build virtual machine with error: %v", returnVMErr)
				}
				getReturnVM = returnVMResult
			}

			updateReturnVM, updateReturnVMErr := machineToVirtualMachine(machine, "SourceTestPvcName")
			if updateReturnVMErr != nil {
				t.Fatalf("Unable to build virtual machine with error: %v", updateReturnVMErr)
			}

			updateReturnVM.Status.Ready = tc.wantVMToBeReady
			mockKubevirtClient.EXPECT().GetVirtualMachine(defaultNamespace, virtualMachine.Name).Return(getReturnVM, tc.clientGetError).AnyTimes()
			mockKubevirtClient.EXPECT().UpdateVirtualMachine(defaultNamespace, virtualMachine).Return(updateReturnVM, tc.clientUpdateError).AnyTimes()
			// TODO: added cases when existingVM == nil :
			// p.machine.Spec.ProviderID != nil && *p.machine.Spec.ProviderID != "" && (p.machine.Status.LastUpdated == nil || p.machine.Status.LastUpdated.Add(requeueAfterSeconds*time.Second).After(time.Now())) - return error
			// else - another error
			kubevirtClientMockBuilder := func(kubernetesClient *kubernetesclient.Clientset, secretName, namespace string) (kubevirtClient.Client, error) {
				return mockKubevirtClient, nil
			}
			providerVMInstance := New(kubevirtClientMockBuilder, nil)

			updateVMErr := providerVMInstance.Update(machine)

			if tc.wantValidateMachineErr != "" {
				assert.Equal(t, tc.wantValidateMachineErr, updateVMErr.Error())
			} else if tc.clientGetError != nil {
				assert.Equal(t, tc.clientGetError.Error(), updateVMErr.Error())
			} else if tc.wantUpdateErr != "" {
				assert.Equal(t, tc.wantUpdateErr, updateVMErr.Error())
			} else if tc.emptyGetVM {
				assert.Equal(t, updateVMErr.Error(), "requeue in: 3m0s")
			} else if !tc.wantVMToBeReady {
				assert.Equal(t, updateVMErr.Error(), "requeue in: 20s")
			} else {
				assert.Equal(t, updateVMErr, nil)
				//providerID := fmt.Sprintf("kubevirt:///%s/%s", machineScope.machine.GetNamespace(), machineScope.virtualMachine.GetName())
				assert.Equal(t, *machine.Spec.ProviderID, tc.providerID)
			}
		})
	}

}

func DefaultVirtualMachine(started bool) (*kubevirtapiv1.VirtualMachine, *kubevirtapiv1.VirtualMachineInstance) {
	return DefaultVirtualMachineWithNames(started, "testvmi", "testvmi")
}

func DefaultVirtualMachineWithNames(started bool, vmName string, vmiName string) (*kubevirtapiv1.VirtualMachine, *kubevirtapiv1.VirtualMachineInstance) {
	vmi := kubevirtapiv1.NewMinimalVMI(vmiName)
	vmi.Status.Phase = kubevirtapiv1.Running
	vm := VirtualMachineFromVMI(vmName, vmi, started)
	t := true
	vmi.OwnerReferences = []metav1.OwnerReference{{
		APIVersion:         kubevirtapiv1.VirtualMachineGroupVersionKind.GroupVersion().String(),
		Kind:               kubevirtapiv1.VirtualMachineGroupVersionKind.Kind,
		Name:               vm.ObjectMeta.Name,
		UID:                vm.ObjectMeta.UID,
		Controller:         &t,
		BlockOwnerDeletion: &t,
	}}
	return vm, vmi
}

func VirtualMachineFromVMI(name string, vmi *kubevirtapiv1.VirtualMachineInstance, started bool) *kubevirtapiv1.VirtualMachine {
	vm := &kubevirtapiv1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: vmi.ObjectMeta.Namespace, ResourceVersion: "1"},
		Spec: kubevirtapiv1.VirtualMachineSpec{
			Running: &started,
			Template: &kubevirtapiv1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   vmi.ObjectMeta.Name,
					Labels: vmi.ObjectMeta.Labels,
				},
				Spec: vmi.Spec,
			},
		},
	}
	return vm
}