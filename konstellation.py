#!/usr/bin/env python3
"""Konstellation main entrypoint."""

from lib.helpers.banner import banner
from lib.helpers.helpers import parse_args, Modes
from lib.helpers.logger import get_logger
from lib.platform.k8s import K8s

def main(config):
    """Main method that calls the appropriate platform feature."""
    platform = K8s(config)

    if config.mode == Modes.enum.name:
        platform.enum()
    elif config.mode == Modes.push.name:
        platform.push()
    elif config.mode == Modes.query.name:
        platform.query()

if __name__ == '__main__':
    print(banner())
    conf = parse_args()
    get_logger(conf)
    main(conf)
