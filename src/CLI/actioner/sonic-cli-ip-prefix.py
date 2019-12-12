#!/usr/bin/python
###########################################################################
#
# Copyright 2019 Dell, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
###########################################################################

import sys
import time
import json
import ast
import cli_client as cc
from rpipe_utils import pipestr
from scripts.render_cli import show_cli_output

import urllib3
urllib3.disable_warnings()

def invoke(func, args):
    body = None
    aa = cc.ApiClient()

    # IPv4 prefix set for exact range (exact)
    if func == 'ipv4_prefix':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV4"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": "exact","config": {
                "ip-prefix": args[2],"masklength-range": "exact","openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # IPv4 prefix set with LE and GE values (le .. ge) 
    elif func == 'ipv4_prefix_le_ge':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV4"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"config": {
                "ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # IPv4 prefix set with only LE (len .. le)   
    elif func == 'ipv4_prefix_le':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV4"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"config": {
                "ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # IPv4 prefix set with only GE (ge .. 32)   
    elif func == 'ipv4_prefix_ge':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV4"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": args[3]+"..32","config": {
                "ip-prefix": args[2],"masklength-range": args[3]+"..32","openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # Remove the ipv4 prefix set.
    elif func == 'ipv4_prefix_delete':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets/prefix-set={prefix_list_name}',
                prefix_list_name=args[0])
        return aa.delete(keypath)

    # IPv6 prefix set for exact range (exact)
    elif func == 'ipv6_prefix':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV6"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": "exact","config": {
                "ip-prefix": args[2],"masklength-range": "exact","openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # IPv6 prefix set with LE and GE values (le .. ge) 
    elif func == 'ipv6_prefix_le_ge':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV6"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"config": {
                "ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # IPv6 prefix set with only LE (len .. le)   
    elif func == 'ipv6_prefix_le':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV6"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"config": {
                "ip-prefix": args[2],"masklength-range": args[3]+".."+args[4],"openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)

    # IPv6 prefix set with only GE (ge .. 128)   
    elif func == 'ipv6_prefix_ge':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets')
        body = {"openconfig-routing-policy:prefix-sets":{"prefix-set":[{"name": args[0],"config":{"name": args[0],
                "mode": "IPV6"},"prefixes":{"prefix":[{"ip-prefix": args[2],"masklength-range": args[3]+"..128","config": {
                "ip-prefix": args[2],"masklength-range": args[3]+"..128","openconfig-routing-policy-ext:action": args[1].upper()}}]}}]}}
        return aa.patch(keypath, body)
    
    # Remove the ipv6 prefix set.
    elif func == 'ipv6_prefix_delete':
        keypath = cc.Path('/restconf/data/openconfig-routing-policy:routing-policy/defined-sets/prefix-sets/prefix-set={prefix_list_name}',
                prefix_list_name=args[0])
        return aa.delete(keypath)

    else:
        return body

def run(func, args):
    try:
        api_response = invoke(func,args)
        return
    except:
            # system/network error
            print "Error: Transaction Failure"



if __name__ == '__main__':

    pipestr().write(sys.argv)
    run(sys.argv[1], sys.argv[2:])

