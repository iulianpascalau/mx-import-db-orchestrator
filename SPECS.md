Orchestration of the import-db process on MultiversX chain. 

The import-db process is the process on which a node's binary, along with its corresponding configuration, is made to 
re-run and reprocess the whole chain's history. It is used to identify and debug backwards compatibility issues or just 
re-generate required storage DB's. Can be used on any network, but it proved great value on the mainnet history.

The data required to re-process everything is made out on the regular node's epoch data, including the transactions storage, 
header and metaheaders storages, smartcontract's storage and so on. Important: it does not need the AccountsDB large storage directory.

Since the MultiversX chain is a sharded chain, the import-db process will need to be run on all 4 shards to fully test the 
backwards compatibility status. Since this is a lengthy process, when checking the backwards compatibility status, the following shortcut can be made:
start multiple containers/VMs that will execute the import-db process on a defined shard on a defined range of epochs. Just like this example:
- VM0: shard 0, epochs 0-100
- VM1: shard 1, epochs 0-80
- VM2: shard 2, epochs 0-100
- VM3: shard metachain, epochs 0-200
- VM4: shard 0, epochs 99-200
- VM5: shard 1, epochs 79-160
- VM6: shard 2, epochs 99-200
- VM7: shard metachain, epochs 199-400

... and so on.

Observe that the ranges must intersect as to fully test all transitions, even on the epochs that are considered the interval ends.
Also, the shard's epoch intervals do not necessarily need to match. 

The solution should be written in Go for the service(s) and in TypeScript/React for the UI.

The following solution will need to accomplish the following tasks:
- [ ] Able to switch on/off a configurable set of Dell PowerEdge servers using the iDRAC API. Also, the status of the running 
server along with the status of running Proxmox hypervisor should be acknowledged. 
- [ ] A comprehensible UI/UX that will show all configured along with the 2 statuses and the possibility to switch on/off the servers.
- [ ] A page showing a list of all VMs/CTs that are configured for the import-db process.






<i>Trademarks: Proxmox, Dell, PowerEdge, iDRAC are trademarks or registered trademarks of their respective owners. Their use does not imply any affiliation with or endorsement by them. </i>