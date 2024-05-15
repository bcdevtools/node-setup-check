## Node setup check
- Validator node
- RPC node
- Snapshot node
- Archival node

```bash
nodesc check ~/.node_home --type validator/rpc/snapshot/archival
```

## Install
```bash
go install github.com/bcdevtools/node-setup-check/cmd/nodesc@latest
```

## Spec
- Check permission of all files
- Check pruning settings
    - [x] Validator node
    - [x] RPC node
    - [x] Snapshot node
    - [x] Archival node
- Check snapshot settings
    - [x] Validator node: should disable
    - [x] Non-validator: should enable
- Check keyring settings
    - [x] Validator node: should have keyring file
    - [x] Non-validator: should not have keyring
- Double sign check
    - [x] Validator node: 10
    - [x] Non-validator node: 0
- P2P port check
    - [x] Should not be default port 26656
- API & Json-RPC settings
    - Validator node
        - [x] Disable API
        - [x] Disable Json-RPC
    - RPC node
        - [x] Enable API
        - [x] Enable Json-RPC
- Check peers config:
    - [x] Maximum inbound & outbound peers, should greater than default
    - [x] Seeds should be set
    - [x] Persistent peers should be set
- Suggest firewall
    - [x] Validator: allow P2P, close other ports
    - [x] RPC: allow P2P, RPC, Rest API, Json-RPC
    - [x] Snapshot: allow P2P, close other ports
    - [x] Archival: allow P2P, RPC, Rest API, Json-RPC
- Check tx index config:
    - [x] Validator: should disable
    - [x] RPC: should enable
    - [x] Archival: should enable
- Check service
    - [x] Do not auto restart
    - [x] Do not enable on boot