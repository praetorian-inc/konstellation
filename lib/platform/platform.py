"""Parent class for all Platforms.

The Platform class implements push and query, but leaves enum
for the individual platforms to implement.
"""

from argparse import Namespace
from colorama import init
from jinja2 import Environment, FileSystemLoader
import json
import logging
from neo4j import GraphDatabase, Query
from neo4j.exceptions import AuthError, ClientError, TransientError, ServiceUnavailable
import os
import re
from lib.helpers.helpers import Modes
import sys
from tqdm import tqdm
from termcolor import colored
import yaml

logger = logging.getLogger('konstellation')

# init colorama
init()

class Platform(object):
    """Platform base class."""
    def __init__(self, config: Namespace, **kwargs) -> None:
        self.config = config

        # assing all kwargs as class members
        for key, value in kwargs.items():
            setattr(self, key, value)

        if self.config.mode in [Modes.push.name, Modes.query.name]:
            # Load and validate mappings
            platform_config_file = os.path.join('resources',
                                                self.name, 'config.yml')
            if not os.path.exists(platform_config_file):
                raise FileNotFoundError(f'{self.name} ' \
                                        f'{platform_config_file} ' \
                                        'config file not found.')

            logger.debug('Loading %s', platform_config_file)
            with open(platform_config_file, 'r', encoding='ascii') as fin:
                platform_config = yaml.safe_load(fin.read())
            self.mappings = platform_config['mappings']
            self.order = platform_config.get('order', {})
            self.relationships = platform_config['relationships']
            self.queries = platform_config['queries']

            if not 'default' in self.mappings:
                raise KeyError(f'{platform_config_file} is missing' \
                               ' a "default" template mapping entry')


            # Set up Jinja2
            self.templates = Environment(
                loader=FileSystemLoader(
                os.path.join('resources', self.name)))

            # Create neo4j driver if we're going to perform a push
            self.neo4j = GraphDatabase.driver(self.config.neo4juri,
                                              auth=(self.config.neo4juser,
                                                    self.config.neo4jpass))
            try:
                self.neo4j.verify_connectivity()
            except (ServiceUnavailable, ConnectionRefusedError):
                logger.error('Neo4j unavailable at %s', self.config.neo4juri)
                sys.exit()
            except AuthError as e:
                logger.error('Neo4j AuthError: %s', e)
                sys.exit()

        if self.config.mode == Modes.query.name:
            self._make_dir_if_not_exist(self.config.results)

    @property
    def name(self) -> str:
        return self.__class__.__name__

    def enum(self) -> None:
        raise NotImplementedError(f'Enum for {self.name} has not' \
                                  f'been implemented yet.')

    def push(self) -> None:
        try:
            results = os.listdir(self.config.enum)
        except FileNotFoundError:
            logger.error('%s not found. Run `%s enum` first,' \
                         ' or specify an alternative directory with `--enum`',
                         self.config.enum, self.name)
            sys.exit()

        if not self.config.relationships:
            logger.warning('Pushing %s data from %s into %s.',
                           self.name,
                           os.path.abspath(self.config.enum),
                           self.config.neo4juri)
            ordered = self._order_items(results)
            for filename in tqdm(ordered):
                # render abs path for result file so neo4j can locate it
                path = os.path.join(os.path.abspath(self.config.enum), filename)
                template_name = self._get_template_name(filename)
                template_file_name = self._get_template_file_name(template_name)

                logger.info('Importing %s using template %s.',
                            filename, template_name)
                if self.mappings[template_name].get('labelField'):
                    label = self.mappings[template_name].get('labelField')
                else:
                    label = f"'{self.mappings[template_name].get('label')}'"

                data = {
                    'path': path,
                    'jsonPath': self.mappings[template_name].get('jsonPath', '$'),
                    'label': label,
                    'labelField': self.mappings[template_name].get('labelField', None),
                    'nameField': self.mappings[template_name].get('nameField', '$') 
                }

                logger.debug('data: %s', data)
                cypher = self._render_template(template_file_name, **data)
                logger.debug(cypher)
                try:
                    self._neo4j_query(cypher)
                except ClientError as e:
                    logger.error('Error importing %s', filename)
                    logger.debug(e)
                except TransientError as e:
                    logger.error(e)

        self._apply_relationship_mappings()

    def query(self) -> None:
        for q in self.queries:
            if q.get('template', None):
                query = self._render_template(q.get('template'))
            else:
                query = q.get('query')

            if self.config.name not in [None ,q.get('name')]:
                continue

            logger.warning('Executing query: %s', q.get('name'))
            res = self._neo4j_query(query)

            filename = self._make_filename(q.get('name'), 'json')
            path = os.path.join(self.config.results, filename)

            data = None
            try:
                data = json.dumps(res, indent=2)
            except TypeError:
                # only fields were returned
                data = str(res)

            if self.config.print:
                for i in res:
                    tqdm.write(str(i))

            if len(res) > 0:
                logger.critical(colored(f'Found {len(res)} results', 'green'))
                logger.warning('Writing %s "%s" results to %s.',
                               len(res), q.get('name'), path)
                with open(os.path.abspath(path), 'w', encoding='ascii') as fout:
                    fout.write(data)
            else:
                logger.warning('Found 0 results')

    def _make_filename(self, value: str, extension: str) -> str:
        """Convert query description to a reasonable file name."""
        value = value.lower()
        value = re.sub(r'[^\w\s-]', '', value)
        value = re.sub(r'[-\s]+', '-', value)
        return f'{value}.{extension}'

    def _neo4j_query(self, cypher: str) -> list:
        """Execute the cypher query."""
        with self.neo4j.session() as session:
            logger.debug(cypher)
            q = Query('') # workaround for LiteralString error
            q.text = cypher

            res = []
            try:
                res = session.run(q)
            except ClientError as e:
                logger.error('Error executing cypher query: %s', e)
                return []

            # Results belong to the session,
            # so create a list of values to return
            return [ dict(i) for i in res]

    def _get_template_name(self, filename: str) -> str:
        if self.mappings.get(filename, None):
            template = filename
        else:
            template = 'default'

        logger.debug('filename: %s, template: %s', filename, template)
        return template

    def _get_template_file_name(self, template_name: str) -> str:
        template = self.mappings.get(template_name).get('template', 'generic.cypher')

        logger.debug('template_name: %s, template: %s', template_name, template)
        return template

    def _make_dir_if_not_exist(self, dir_: str) -> None:
        if not os.path.exists(dir_):
            os.makedirs(dir_)

    def _render_template(self, template_name: str, **kwargs) -> str:
        template = self.templates.get_template(template_name)
        rendered = template.render(**kwargs)
        return rendered

    def _order_items(self, items: list) -> list:

        if self.order.get('last', None):
            items = [item for item in items if item not in self.order['last']]
            items = items + self.order['last']
        return items

    def _apply_relationship_mappings(self) -> None:
        logger.info('Applying relationships, this may take a while.')
        query = ''
        relationships = []

        if self.config.relationship_name:
            for r in self.relationships:
                if self.config.relationship_name == r['name']:
                    relationships.append(r)
                    break
            if len(relationships) != 1:
                logger.error('Relationship %s not found.', self.config.relationship_name)
                sys.exit()
        else:
            relationships = self.relationships

        for r in tqdm(relationships):
            data = {}
            if r.get('results_file'):
                path = os.path.join(os.path.abspath(self.config.enum), r['results_file'])
                data = {'path': path}

            query = self._render_template(r['template'], **data)
            logger.warning('Applying %s - %s', r['name'], r.get('description', ''))
            self._neo4j_query(query)
