/* Released under public domain */

package main

import (
	"context"
	"log"
	"fmt"
	"net/url"
	"os"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

func main() {
	ctx := context.Background()
	
	// Create vSphere API client
	u, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalf("url.Parse(): %v", err)
	}
	u.User = url.UserPassword(os.Getenv("VC_USERNAME"), os.Getenv("VC_PASSWORD"))

	c, err := govmomi.NewClient(ctx, u, /* insecure */ false)
	if err != nil {
		log.Fatalf("NewClient(): %v", err)
	}
	
	// Locate datacenter to use and select as default
	finder := find.NewFinder(c.Client, /* recurse all */ false)
	dc, err := finder.Datacenter(ctx, "bogal")
	if err != nil {
		log.Fatalf("find.DefaultDatacenter(): %v", err)
	}
	finder.SetDatacenter(dc)

	folders, err := dc.Folders(ctx)
	if err != nil {
		log.Fatalf("Folders(): %v", err)
	}

	// Create VM
	spec := &types.VirtualMachineConfigSpec{
		Name:       "test-vm",
		GuestId:    "otherGuest",
		NumCPUs:    int32(1),
		MemoryMB:   int64(1024),
		Annotation: "Test annotation",
		Firmware:   string(types.GuestOsDescriptorFirmwareTypeEfi),
	}

	// Create disk controller
	var devices object.VirtualDeviceList
	controller, err := devices.CreateSCSIController("pvscsi")
	if err != nil {
		log.Fatalf("CreateSCSIController(pvscsi): %v", err)
	}
	devices = append(devices, controller)

	// Create disk
	disk :=  &types.VirtualDisk{
		VirtualDevice: types.VirtualDevice{
			Key: devices.NewKey(),
			Backing: &types.VirtualDiskFlatVer2BackingInfo{
				DiskMode:        string(types.VirtualDiskModePersistent),
				ThinProvisioned: types.NewBool(true),
			},
		},
		CapacityInKB: 1024*1024,
	}
	
	// Attach disk to controller
	devices.AssignController(disk, controller.(types.BaseVirtualController))
	devices = append(devices, disk)

	// Create network interface vmxnet3, non-DVS, using the network "VM Network"
	backing := &types.VirtualEthernetCardNetworkBackingInfo{
		VirtualDeviceDeviceBackingInfo: types.VirtualDeviceDeviceBackingInfo{
			VirtualDeviceBackingInfo: types.VirtualDeviceBackingInfo{},
			DeviceName:               "VM Network",
			UseAutoDetect:            types.NewBool(false),
		},
		Network:           (*types.ManagedObjectReference)(nil),
		InPassthroughMode: types.NewBool(false),
	}
	netdev, err := object.EthernetCardTypes().CreateEthernetCard("vmxnet3", backing)
	if err != nil {
		log.Fatalf("CreateEthernetCard(vmxnet3, VM Network): %v", err)
	}
	devices = append(devices, netdev)

	// Finish the VM configuration
	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		log.Fatalf("ConfigSpec(VirtualDeviceConfigSpecOperationAdd): %v", err)
	}

	// Store in datastore1
	datastore, err := finder.Datastore(ctx, "datastore1")
	if err != nil {
		log.Fatalf("finder.Datastore(datastore1): %v", err)
	}

	spec.DeviceChange = deviceChange
	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", datastore.Name()),
	}

	// Use default resource pool for this datacenter
	rp, err := finder.DefaultResourcePool(ctx)
	if err != nil {
		log.Fatalf("DefaultResourcePool(): %v", err)
	}
	
	// Execute
	_, err = folders.VmFolder.CreateVM(ctx, *spec, rp, nil)
	if err != nil {
		log.Fatalf("CreateVM(): %v", err)
	}
}
