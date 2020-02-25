import argparse
import sys
import limacharlie
import json
import hmac
import hashlib
import urllib.request
import urllib.error
import urllib.parse
import time
import uuid

def main():
    parser = argparse.ArgumentParser( prog = 'python -m lcservice.simulator' )
    parser.add_argument( 'url',
                         type = str,
                         help = 'the URL of the service to call.' )
    parser.add_argument( 'action',
                         type = str,
                         help = 'action to run, one of supported callbacks (see doc).' )
    parser.add_argument( '-s', '--secret',
                         type = str,
                         required = False,
                         dest = 'secret',
                         default = '',
                         help = 'optionally specify the shared secret, you can omit if the service was started with secret disabled.' )
    parser.add_argument( '-d', '--data',
                         type = eval,
                         default = {},
                         dest = 'data',
                         help = 'data to include in the request, this string gets evaluated as python code' )

    args = parser.parse_args()

    if args.url.startswith( 'https://' ) and args.url.startswith( 'http://' ):
        print( "url must start with http or https." )
        sys.exit( 1 )

    lc = limacharlie.Manager()
    lc.whoAmI()
    oid = lc._oid
    jwt = lc._jwt
    secret = args.secret

    print( json.dumps( postData( args.url, secret, oid, jwt, args.action, data = args.data ), indent = 2 ) )

def postData( dest, secret, oid, jwt, eventType, data = {} ):
    data = {
        'data' : data,
    }
    data[ 'oid' ] = oid
    data[ 'deadline' ] = int( time.time() ) + 590
    data[ 'mid' ] = str( uuid.uuid4() )
    data[ 'etype' ] = eventType
    data[ 'jwt' ] = jwt
    data = json.dumps( data, sort_keys = True ).encode()
    headers = {
        'lc-svc-sig' : hmac.new( secret.encode(), msg = data, digestmod = hashlib.sha256 ).hexdigest(),
        'User-Agent' : 'lc-services',
        'Content-Type' : 'application/json',
    }
    resp = urllib.request.urlopen( urllib.request.Request( dest, data, headers ), timeout = 60 )
    resp = json.loads( resp.read() )
    return resp

if __name__ == '__main__':
    main()