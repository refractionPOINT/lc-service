import lcservice
import uuid

TEST_SECRET = 'test-secret'

def test_create_service():
    svc = lcservice.Service( 'test-service', None )

    svc.setRequestParameters( {
        "action": {
            "type": "str",
            "desc": "the action to take, one of 'set' or 'get'.",
        },
        "api_key": {
            "type": "str",
            "desc": "the api key to use when requesting a payload detonation."
        },
        "retention": {
            "type": "int",
            "desc": "the number of days to set when ingesting detonation artifacts."
        }
    } )

    resp = svc._processEvent( {
        'etype' : 'health',
    } )

    assert( resp )
    data = resp[ 'data' ]
    isSuccess = resp[ 'success' ]
    assert( data )
    assert( isSuccess )
    assert( 'version' in data )
    assert( 'start_time' in data )
    assert( 0 == len( data[ 'mtd' ][ 'detect_subscriptions' ] ) )
    assert( 2 == len( data[ 'mtd' ][ 'callbacks' ] ) )
    assert( 3 == len( data[ 'mtd' ][ 'request_params' ] ) )

def test_callback_enabled():
    class _Svc( lcservice.Service ):
        def onOrgInstalled( self, lc, oid, request ):
            return self.response( isSuccess = True, data = { 'test' : 42 } )

    svc = _Svc( 'test-service', None )

    resp = svc._processEvent( {
        'etype' : 'health',
    } )

    assert( resp )
    data = resp[ 'data' ]
    isSuccess = resp[ 'success' ]
    assert( data )
    assert( isSuccess )
    assert( 'version' in data )
    assert( 'start_time' in data )
    assert( 0 == len( data[ 'mtd' ][ 'detect_subscriptions' ] ) )
    assert( 3 == len( data[ 'mtd' ][ 'callbacks' ] ) )

    resp = svc._processEvent( {
        'etype' : 'org_install',
        'oid' : str( uuid.uuid4() ),
    } )

    assert( resp )
    data = resp[ 'data' ]
    isSuccess = resp[ 'success' ]
    assert( data )
    assert( isSuccess )
    assert( data[ 'test' ] == 42 )

if __name__ == '__main__':
    test_create_service()