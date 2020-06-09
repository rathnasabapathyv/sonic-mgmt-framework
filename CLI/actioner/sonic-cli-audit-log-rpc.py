from collections import OrderedDict
import sys
import json
from rpipe_utils import pipestr
from scripts.render_cli import show_cli_output
import cli_client as cc

def run(func, args):
    aa = cc.ApiClient()

    if func == 'rpc_sonic_auditlog_show_auditlog':
        keypath = cc.Path('/restconf/operations/sonic-auditlog:show-auditlog')
        if (itype == "all"):
            body = { "sonic-auditlog:input": {"content-type" : "all"} }
        else:
            body = { "sonic-auditlog:input": {"content-type" : "brief"} }
        response = aa.post(keypath, body)
        if response.ok():
            print(response.content["sonic-show-auditlog:output"]["audit-content"])
        else:
            print "%Error: Transaction Failure "
            print(response.error_message())
            print(response.status_code)

    if func == 'rpc_sonic_auditlog_clear_auditlog':
        keypath = cc.Path('/restconf/operations/sonic-auditlog:clear-auditlog')
        body = { "sonic-auditlog:input": "" }
        response = aa.post(keypath, body)
        if response.ok():
            print("%Success: audit log is cleared.")
        else:
            print "%Error: Transaction Failure "
            print(response.error_message())
            print(response.status_code)

if __name__ == '__main__':

    pipestr().write(sys.argv)
    if (len(sys.argv) == 2) :
        func = sys.argv[1]
        itype = "brief"
        run(func, sys.argv[1:])
    else:
        itype = sys.argv[1]
        func = sys.argv[2]
        run(func, sys.argv[2:])

