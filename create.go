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

	u, err := url.Parse(os.Args[1])
	if err != nil {
		log.Fatalf("url.Parse(): %v", err)
	}
	u.User = url.UserPassword(os.Getenv("VC_USERNAME"), os.Getenv("VC_PASSWORD"))

	c, err := govmomi.NewClient(ctx, u, /* insecure */ false)
	if err != nil {
		log.Fatalf("NewClient(): %v", err)
	}
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

	spec := &types.VirtualMachineConfigSpec{
		Name:       "test-vm",
		GuestId:    "otherGuest",
		NumCPUs:    int32(1),
		MemoryMB:   int64(1024),
		Annotation: "Test annotation",
		Firmware:   string(types.GuestOsDescriptorFirmwareTypeEfi),
	}

	var devices object.VirtualDeviceList
	controller, err := devices.CreateSCSIController("pvscsi")
	if err != nil {
		log.Fatalf("CreateSCSIController(pvscsi): %v", err)
	}
	devices = append(devices, controller)

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
	devices.AssignController(disk, controller.(types.BaseVirtualController))
	devices = append(devices, disk)

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

	deviceChange, err := devices.ConfigSpec(types.VirtualDeviceConfigSpecOperationAdd)
	if err != nil {
		log.Fatalf("ConfigSpec(VirtualDeviceConfigSpecOperationAdd): %v", err)
	}

	datastore, err := finder.Datastore(ctx, "datastore1")
	if err != nil {
		log.Fatalf("finder.Datastore(datastore1): %v", err)
	}

	spec.DeviceChange = deviceChange
	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: fmt.Sprintf("[%s]", datastore.Name()),
	}

	rp, err := finder.DefaultResourcePool(ctx)
	if err != nil {
		log.Fatalf("DefaultResourcePool(): %v", err)
	}
	_, err = folders.VmFolder.CreateVM(ctx, *spec, rp, nil)
	if err != nil {
		log.Fatalf("CreateVM(): %v", err)
	}
}
