import lcservice
import uuid
import time

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

def test_parameters():
    class _Svc( lcservice.Service ):
        def onStartup( self ):
            self.setRequestParameters( {
                'action' : {
                    'type' : 'enum',
                    'values' : [
                        'cmd1',
                        'cmd2',
                    ],
                    'desc' : 'ccc',
                    'is_required' : True,
                },
                'artifact_id' : {
                    'type' : 'str',
                    'desc' : 'bbb.',
                    'is_required' : True,
                },
                'retention' : {
                    'type' : 'int',
                    'desc' : 'aaa.',
                },
            } )

        def onRequest( self, lc, oid, request ):
            return True

    svc = _Svc( 'test-service', None )

    # Missing required parameter.
    resp = svc._processEvent( {
        'etype' : 'request',
        'data' : {},
    } )

    assert( not resp[ 'success' ] )

    # Valid partial.
    resp = svc._processEvent( {
        'etype' : 'request',
        'data' : {
            'action' : 'cmd1',
            'artifact_id' : 'zzz'
        },
    } )

    assert( resp[ 'success' ] )

    # Invalid type.
    resp = svc._processEvent( {
        'etype' : 'request',
        'data' : {
            'action' : 'cmd1',
            'artifact_id' : 33
        },
    } )

    assert( not resp[ 'success' ] )

    # Invalid enum.
    resp = svc._processEvent( {
        'etype' : 'request',
        'data' : {
            'action' : 'nope',
            'artifact_id' : 'zzz'
        },
    } )

    assert( not resp[ 'success' ] )

    # Extra parameters.
    resp = svc._processEvent( {
        'etype' : 'request',
        'data' : {
            'action' : 'cmd1',
            'artifact_id' : 'zzz',
            'ccc' : 'yes'
        },
    } )

    assert( resp[ 'success' ] )

if __name__ == '__main__':
    test_create_service()


n = 0
def test_schedules():
    global n
    svc = lcservice.Service( 'test-service', None )

    def _inc():
        global n
        n += 1

    svc.delay( 5, _inc )
    assert( 0 == n )
    time.sleep( 6 )

    n = 0

    svc.schedule( 2, _inc )
    assert( 1 == n )
    time.sleep( 2.1 )
    assert( 2 == n )
    time.sleep( 2.1 )
    assert( 3 == n )

    svc._onShutdown()