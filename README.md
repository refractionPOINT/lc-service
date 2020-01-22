# lc-service
Reference implementation of the LimaCharlie Service protocol.

# Reference Implementation
LimaCharlie services implement a publicly documented JSON/HTTP based protocol.
This means you may implement a service that speaks this protocol in whaterver
language or architecture you like.

That being said, to simplify the life of users and to describe more concretely
the expectation of a working service, LimaCharlie publishes an open source
Reference Implementation (RI) in Python.

## Transports
The RI supports two transports out of the box: standalone container based on
the CherryPy project for an HTTP server, and a Google Cloud Function compatible
server to deploy without containers or infrastructure. Writing transports is easy
so feel free to suggest new ones.

## Using

## Deploying

# Protocol
LimaCharlie Services rely entirely on response to REST calls (webhooks)
from LimaCharlie, making passive deployments through AWS Lambda, GCP
Cloud Functions or GCP Cloud Run possible.

Each HTTP POST contains JSON. The request content and responses are entirely
based on JSON content and not HTTP status codes (to possibly enable other
transport protocols if needed in the future).

A request contains the following data:

* `version`: this is the version of the protocol spoken by LimaCharlie.
* `oid`: the Organization ID this call is about (if any).
* `mid`: a unique Message ID which can be used to perform idempotent operations.
* `deadline`: a timestamp of how long LimaCharlie is willing to wait for this call.
* `jwt`: a JWT for the given `oid`, valid for AT LEAST 30 minutes, with the requester permissions for the service.
* `etype`: an event type (described below).
* `data`: arbitrary JSON, content depends on the `etype`.

A response from the service, also JSON is expected to have the following format:

* `success`: a boolean indicating whether the call was successful.
* `retry`: if `success` was `false`, should LimaCharlie attempt to re-deliver this message.
* `data`: arbitrary JSON, content based on the `etype` in the request.

Most requests will have a deadline of +60s in the future. This may mean that longer
operations will not fit in that deadline. You should either delay execution, parallelize
or split up the execution in more granular `etype` events like per-sensor events.

More complex `etype` will be available in the future to allow services to request
extensions or record longer running jobs.

## Event Types

### health
This call is LimaCharlie requesting a status from your service. The following information
is expected to be returned in the data.

* `version`: the version of the protocol spoken by the service.
* `start_time`: a timestamp of when this service instance started.
* `calls_in_progress`: the number of of calls in progress on this instance.
* `mtd`: a JSON dictionary with the following keys descibed below.

**Metadata** found in the `mtd` key describes what subset of the protocol is
used by this service:

* `detect_subscriptions`: a list of detections this service would like to receive for organizations subscribed.
* `callbacks`: the list of `etypes` supported/used by this service (telling LimaCharlie not to bother with the others).

### org_install
Indicates that a new organization has installed the service (subscribed).
Setup the organization with all the required configurations here.

### org_uninstall
Indicates that an organization has uninstalled the service. Remove all configurations
that were made on that organization here. All traces of your service should be gone.

### detection
Called when a detection this service subscribes to (see `detect_subscriptions`) occurs.

### request
Will support interactive requests by users within the organization for ad-hoc functionality
of your service like running jobs on specific hosts etc. Not yet implemented.

### org_per_*
Convenience cron-like event. The LimaCharlie cloud emits those events at recurring
interval on a per-organization basis so you don't have to keep track of timing or
setup cron jobs.

* `org_per_1h`
* `org_per_3h`
* `org_per_12h`
* `org_per_24h`
* `org_per_7d`
* `org_per_30d`

### once_per_*
Convenience cron-like event. The LimaCharlie cloud emits those events at recurring
interval on a per-service basis so you don't have to keep track of timing or
setup cron jobs.

* `once_per_1h`
* `once_per_3h`
* `once_per_12h`
* `once_per_24h`
* `once_per_7d`
* `once_per_30d`

### sensor_per_*
Convenience cron-like event. The LimaCharlie cloud emits those events at recurring
interval on a per-sensor basis so you don't have to keep track of timing or
setup cron jobs.

* `sensor_per_1h`
* `sensor_per_3h`
* `sensor_per_12h`
* `sensor_per_24h`
* `sensor_per_7d`
* `sensor_per_30d`

### new_sensor
Called when a sensor enrolls in an organization with the service installed.
The `data` component will contain basic information on the new sensor like
a `sid`, `hostname`, `ext_ip` and `int_ip` as well as platform and architecture.
