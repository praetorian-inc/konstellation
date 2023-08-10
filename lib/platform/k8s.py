"""Kubernetes Platform implementation"""

import json
from lib.helpers.helpers import Modes
from lib.platform.platform import Platform
import logging
from tqdm import tqdm
import os



from kubernetes import config as k8s_config
from kubernetes.dynamic import DynamicClient
from kubernetes.dynamic.exceptions import NotFoundError, MethodNotAllowedError, ServiceUnavailableError
from kubernetes.client import api_client
from kubernetes.dynamic.resource import Resource


logger = logging.getLogger('konstellation')

# Using k8s for internal class names to avoid collision with the sdk
class K8s(Platform):
    """Kubernetes Platform implementation."""
    def __init__(self, config, **kwargs):
        super().__init__(config, **kwargs)

        if config.mode == Modes.enum.name:
            # Load kubeconfig
            if config.kubeconfig:
                k8s_config.load_kube_config(config_file=config.kubeconfig)
            elif config.incluster:
                k8s_config.load_incluster_config()
            else:
                k8s_config.load_kube_config()

        # load stanard subresources for comparison when enumerating
        subresources_path = os.path.join('resources',
                                         self.name,
                                         'standard_subresources.json')

        with open(subresources_path, 'r', encoding='ascii') as fin:
            self.subresources = json.loads(fin.read())

    def _remove_secret_data(self, items):
        """ Remove secret details """

        for i in items['items']:
            i['data'] = {}

        return items

    def enum(self) -> None:
        self._make_dir_if_not_exist(self.config.enum)

        resources_count = 0
        resources_retrieved = 0
        client = DynamicClient(api_client.ApiClient())

        for res_list in tqdm(client.resources):  # enumerated resource types
            for r in res_list:  # each type is a list with a single member

                # does not work with isinstance,
                # using type() instead
                if type(r) == Resource:
                    logger.debug(r.to_dict())
                    resources_count += 1

                    if r.group:
                        rname = f'{r.name}.{r.group}'
                    else:
                        rname = f'{r.name}'

                    logger.debug(rname)

                    try:
                        items = r.get().to_dict()
                    except NotFoundError:
                        logger.error('Failed to fetch %s', rname)
                        continue
                    except MethodNotAllowedError:
                        logger.error('List method not allowed on %s', rname)
                        continue
                    except ServiceUnavailableError as e:
                        logger.error('Error requesting %s, ' \
                                     'Service Unavailable, %s',
                                     rname, e)

                    # Remove the secret data
                    if rname == 'secrets':
                        items = self._remove_secret_data(items)

                    filename = os.path.join(self.config.enum, rname + '.json')
                    with open(filename, 'w', encoding='ascii') as fout:
                        fout.write(json.dumps(items, indent=2))


                    if r.subresources:
                        logger.debug(str(r.subresources))
                        for sub in r.subresources.keys():
                            if not r.name in self.subresources.keys():
                                logger.warning(
                                    'Non-standard subresource found: %s - %s',
                                    r.name, sub)
                            else:
                                for verb in r.subresources[sub].verbs:
                                    if verb not in self.subresources[r.name].get(sub, []):
                                        logger.warning(
                                            'Non-standard subresource verb found: %s - %s - %s',
                                                       r.name, sub, verb)


                    resources_retrieved += 1

        logger.warning('Found %s resource types.', resources_count)
        logger.warning('Retrieved %s resource types.', resources_retrieved)
