import uuid
import time
import yaml
import json

class Job( object ):

    def __init__( self, jobId = None ):
        self._isNew = False
        self._json = {}
        if jobId is None:
            self._isNew = True
            jobId = str( uuid.uuid4() )
            self._json[ 'start' ] = int( time.time() )
        self._json[ 'id' ] = jobId

    def getId( self ):
        return self._json[ 'id' ]

    def addSensor( self, sid ):
        self._json.setdefault( 'sid', [] ).append( str( sid ) )

    def setCause( self, cause ):
        self._json[ 'cause' ] = cause

    def close( self ):
        self._json[ 'end' ] = int( time.time() * 1000 )

    def toJson( self ):
        if self._isNew and 'cause' not in self._json:
            raise Exception( '"cause" is required for new jobs' )
        return self._json

    def narrate( self, message, attachments = [], isImportant = False ):
        self._json.setdefault( 'hist', [] ).append( {
            'ts' : int( time.time() * 1000 ),
            'msg' : str( message ),
            'attachments' : [ a.toJson() for a in attachments ],
            'is_important' : bool( isImportant ),
        } )

    def __str__( self ):
        return json.dumps( self.toJson(), indent = 2 )

    def __repr__( self ):
        return json.dumps( self.toJson(), indent = 2 )

class HexDump( object ):
    def __init__( self, caption, data ):
        self._data = {
            'att_type' : 'hex_dump',
            'caption' : str( caption ),
            'data' : data,
        }

    def toJson( self ):
        return self._data

class Table( object ):
    def __init__( self, caption, headers, rows = [] ):
        if not isinstance( headers, ( list, tuple ) ):
            raise Exception( "Table headers must be a list or tuple, not %s" % ( type( headers ), ) )
        self._data = {
            'att_type' : 'table',
            'caption' : str( caption ),
            'headers' : headers,
            'rows' : [],
        }
        for r in rows:
            self.addRow( r )

    def addRow( self, fields ):
        if not isinstance( fields, ( list, tuple ) ):
            raise Exception( "Table row must be list or tuple, not %s" % ( type( fields ), ) )
        self._data[ 'rows' ].append( fields )

    def length( self ):
        return len( self._data[ 'rows' ] )

    def toJson( self ):
        return self._data

class YamlData( object ):
    def __init__( self, caption, data ):
        self._data = {
            'att_type' : 'yaml',
            'caption' : caption,
            'data' : yaml.safe_dump( data, default_flow_style = False ),
        }

    def toJson( self ):
        return self._data

class JsonData( object ):
    def __init__( self, caption, data ):
        self._data = {
            'att_type' : 'json',
            'caption' : caption,
            'data' : json.dumps( data, indent = 2 ),
        }

    def toJson( self ):
        return self._data