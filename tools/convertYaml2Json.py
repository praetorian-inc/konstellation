#!/usr/bin/env python3

import json
import yaml
import os
import sys

d = sys.argv[1]
files = os.listdir(d)
for f in files:
    full = os.path.join(d, f)
    ext = os.path.splitext(full)[1]
    if ext in ['.yml', '.yaml']:
        print(f'Converting {full}')
        y = yaml.safe_load(open(full, 'r').read())
        j = json.dumps(y, indent=2)
        with open(full.replace(ext, '.json'), 'w') as fout:
            fout.write(j)
    else:
        continue
    


