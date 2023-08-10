"""Global constants and argument parsing."""

import argparse
from argparse import Namespace
from enum import Enum
from lib.version import __version__
import sys

# Define mode and platform as enums for easy refernce
# Platform names need to match the class names
# TODO: figure out how to tightly couple this relation
Platforms = Enum('Platform', ['k8s'])
Modes = Enum('Mode', ['enum', 'push', 'query'])

def parse_args() -> Namespace:
    ap = argparse.ArgumentParser(description=f'Konstellation v{__version__}')
    ap.add_argument('platform',
                    help='Platform to enumerate',
                    choices=[member.name for member in Platforms])
    ap.add_argument('mode',
                    help='Operation mode',
                    choices=[member.name for member in Modes])
    ap.add_argument('--debug',
                    '-d',
                    action='store_true',
                    help='Debug output')
    ap.add_argument('--enum',
                    '-e',
                    type=str,
                    help='Enum platform output directory',
                    default=None)
    ap.add_argument('--verbose',
                    '-v',
                    action='store_true',
                    help='Verbose output')
    ap.add_argument('--kubeconfig',
                    '-k',
                    type=str,
                    help='Kubeconfig file',
                    default=None)
    ap.add_argument('--incluster',
                    '-l',
                    type=str,
                    help='Incluster kubernetes',
                    default=None)
    ap.add_argument('--name',
                    '-n',
                    type=str,
                    help='Name of query to run')
    ap.add_argument('--print',
                    '-p',
                    action='store_true',
                    help='Print query results to stdout')
    ap.add_argument('--results',
                    '-r',
                    type=str,
                    help='Query results directory',
                    default=None)
    ap.add_argument('--relationships',
                    action='store_true',
                    help='Only run relationship mappings',
                    default=False)
    ap.add_argument('--relationship-name',
                    type=str,
                    help='Run the named relationship name mapping',
                    default=False)
    ap.add_argument('--neo4juri',
                    type=str,
                    help='neo4j URI',
                    default='bolt://localhost:7687')
    ap.add_argument('--neo4juser',
                    type=str,
                    help='neo4j user',
                    default='neo4j')
    ap.add_argument('--neo4jpass',
                    type=str,
                    help='neo4j password',
                    default='neo4j')

    args = ap.parse_args()

    if args.mode in [Modes.enum.name,  Modes.push.name] and args.enum is None:
        args.enum = f'{args.platform}-enum'
    elif args.mode == Modes.query.name and args.results is None:
        args.results = f'{args.platform}-results'

    if (args.relationships or args.relationship_name) and (
        not args.mode == Modes.push.name):

        sys.exit('--relationships can only be used with push')

    return args
