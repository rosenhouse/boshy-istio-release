# prototype bosh integration for istio

Adds Envoy to each BOSH VM, deploys an Istio Pilot to coordinate them.

- [status](#status)
- [tire kicking](#tire-kicking)
- [behind the scenes](#behind-the-scenes)


## status

#### works
- Istio Pilot adapter for BOSH will pick up special `expose_port` declarations on your BOSH manifest (see [`sample-bosh-manifest.yml`](sample-bosh-manifest.yml))
- BOSH addon deploys Envoy and VIP/DNS agent to every VM
- VIP/DNS agent resolves `*.boshy` names to a virtual IP `169.254.255.*`
- DNAT rule for egress directs all VIP connections into local Envoy proxy
- Envoy will proxy for known `expose_port`s and load-balance across destinations

#### todo
- DNAT rule for ingress (supports advanced features)
- Expose pilot config somehow (k8s apiserver?) and demo route-rule use cases
- integration with Istio CA for transparent mTLS
- integration with Mixer


## tire kicking

#### deploy bosh-istio
```
bosh_login lite   # or otherwise target your director

bosh upload-release releases/boshy-istio/boshy-istio-0.2.0.yml

# configure Istio using credentials in the BOSH_ env vars
bosh -d bosh-istio deploy \
  --var=bosh_director_client_secret="${BOSH_CLIENT_SECRET}" \
  --var-file=bosh_director_ca_cert=<(echo "${BOSH_CA_CERT}") \
  bosh-istio-deployment.yml
```

#### install runtime config
this will provide the Envoy proxy as a BOSH add-on for all jobs

some hackery to get release version:
```
RELEASE_VERSION="$(bosh int --path /releases/name=istio/version <(bosh -d bosh-istio manifest) )"
bosh update-runtime-config --var=release_version=$RELEASE_VERSION runtime-config.yml
```

#### deploy a sample bosh manifest
it will pick up the runtime-config
```
bosh -d sample-deployment deploy sample-bosh-manifest.yml
```

#### make requests
```
bosh -d sample-deployment ssh apricot/3
```

connect to a service by name and port:
```
curl -v http://example-httpbin.banana.sample-deployment.boshy:9002/get
```
note that the name resolves to IP 169.254.255.x


## behind the scenes

observe the known hostnames
```
tail -f /var/vcap/sys/log/virtual-ip-agent/*.log
```

inspect the envoy admin endpoint
```
curl http://127.0.0.1:8001
```

query the istio pilot endpoints
```
export PILOT_URL=http://10.244.3.125:8080
curl $PILOT_URL/v1/listeners/x/sidecar~10.244.0.130~x~x
curl $PILOT_URL/v1/clusters/x/sidecar~10.244.0.130~x~x
curl $PILOT_URL/v1/registration
```
