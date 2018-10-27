# vsphere-go-test

To create a VM named "test-vm":

```
go get
go build
read VC_PASSWORD
# Enter password
export VC_PASSWORD
export VC_USERNAME=administrator@vsphere.local
./vsphere-go-test https://vc.my.domain/sdk
```
