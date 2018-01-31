# prototype bosh integration for istio

### deploy bosh-istio
```
bosh_login lite   # or otherwise target your director

bosh create-release  # this will take a long time, because it runs dep ensure

bosh upload-release

# configure Istio using credentials in the BOSH_ env vars
bosh -d bosh-istio deploy \
  --var=bosh_director_client_secret="${BOSH_CLIENT_SECRET}" \
  --var-file=bosh_director_ca_cert=<(echo "${BOSH_CA_CERT}") \
  bosh-istio-deployment.yml
```

### install runtime config
this will provide the Envoy proxy as a BOSH add-on for all jobs

some hackery to get release version:
```
RELEASE_VERSION="$(bosh int --path /releases/name=istio/version <(bosh -d bosh-istio manifest) )"
bosh update-runtime-config --var=release_version=$RELEASE_VERSION runtime-config.yml
```

### deploy a bosh manifest
this way they pick up the runtime config and will get the add-on
```
bosh -d sample-deployment deploy sample-bosh-manifest.yml
```

```
bosh -d sample-deployment ssh apricot/3
```

on the bosh vm, connect via a virtual-IP to a service:
```
curl http://169.254.255.2:9002/headers | json_pp
```


### query pilot

```
export PILOT_URL=http://10.244.3.125:8080
```

- LDS
  ```
  curl $PILOT_URL/v1/listeners/x/sidecar~10.244.0.130~x~x
  ```

- CDS
  ```
  curl $PILOT_URL/v1/clusters/x/sidecar~10.244.0.130~x~x
  ```

- SDS list all
  ```
  curl $PILOT_URL/v1/registration
  ```

- see virtual IP address assignment
  ```
  curl -s 10.244.3.125:8080/v1/listeners/x/sidecar~10.244.3.125~x~x | grep -C 2 169
  ```
