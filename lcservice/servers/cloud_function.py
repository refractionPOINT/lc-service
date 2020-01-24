import flask
import json

def ServeCloudFunction( service, request ):
    '''Serve a service using a google cloud function.'''

    # Make sure we only handle JSON.
    requestData = request.get_json()
    if requestData is None:
        return flask.Response( 'request must be JSON', status = 400 )

    # Verify the signature.
    sig = request.headers.get( 'lc-svc-sig', '' )
    if not service._verifyOrigin( requestData, sig ):
        return flask.Response( 'bad origin signature', status = 401 )

    # Actually execute the service on the data.
    response = service._processEvent( requestData )

    # Serialize the JSON response.
    return json.dumps( response )