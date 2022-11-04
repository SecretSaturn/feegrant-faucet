# Secret Feegrant Faucet

This faucet app allows anyone to request a fee grant for a Secret account addresses.

## How to deploy a faucet

1. Clone this repository locally

2. Install [secretcli](https://github.com/scrtlabs/SecretNetwork/releases) on the server. `secretcli`'s version has to be compatible with the mainnet.

3. Create the the faucet account on the machine that is going to run the faucet. Add the --keyring-backend== command if you wish to use a different keybackend. By default the faucet uses --keyring-backend==test.
    ```
    secretcli keys add <name of the account> (--keyring-backend==test)
    ```

4. Make sure the faucet account has funds. The faucet basically performs a `tx feegrant grant` for every token request, so make sure the faucet account has enough tokens (more tokens could be added later by sending more funds to the faucet account).

5. Copy the `.env` template to the `/frontend` directory and change the `.env` parameters as you see fit.
    ```
    cp .env.template ./frontend/.env
    ```
    
6. Build:
    ```
    make all
    ```

7. Deploy to server. You can do it manually by copying the `bin/` directory or run `make deploy` (make sure to change the makefile to match your server's address i.e. `scp -r ./bin user-name@your.domain:~/`)


8. (optional) You can start the server by running the `./path/to/bin/faucet` binary. It is recommended to create a systemd unit. For example (change parameters for your own deployment):
    ```
    [Unit]
    Description=Faucet web server
    After=network.target

    [Service]
    Type=simple
    WorkingDirectory=/home/ubuntu/feegrant-faucet/bin
    ExecStart=/home/ubuntu/feegrant-faucet/bin/faucet
    User=ubuntu
    Restart=always
    StartLimitInterval=0
    RestartSec=3
    LimitNOFILE=65535
    AmbientCapabilities=CAP_NET_BIND_SERVICE

    [Install]
    WantedBy=multi-user.target
    ```
