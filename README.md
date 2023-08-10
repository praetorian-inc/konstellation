# Konstellation

Konstellation is a configuration-driven CLI tool to enumerate cloud resources and store the data into neo4j.

# Installation

## Python
Konstellation is a Python3 application and can have its dependencies installed using the following commands.

```bash
python3 -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

## Neo4j

Konstellation uses Neo4j as its backend database. [Neo4j Desktop](https://neo4j.com/download/) is the preferred installation method for Neo4j. Installation instructions are [here](https://neo4j.com/docs/desktop-manual/current/installation/download-installation/). 

After installing Neo4j Desktop, create a new project for Konstellation to house the database and configuration settings. When [creating a new database](https://neo4j.com/developer/neo4j-desktop/#desktop-create-DBMS), use a 4.x version that is greater than or equal to 4.4.

After creating the DBMS, enable the [APOC]() library according to these [instructions](https://neo4j.com/labs/apoc/4.1/installation/#neo4j-desktop). Konstellation uses APOC to enable the direct processing and conversion of JSON to nodes and relationships.

Using the [DBMS settings](https://neo4j.com/labs/apoc/4.1/installation/#neo4j-desktop), add the following configuration directives to provide Konstellation sufficient privileges.

```ini
dbms.security.procedures.allowlist=apoc.convert.getJsonProperty,apoc.convert.toJson,apoc.convert.toList,apoc.create.setProperties,apoc.create.setProperty,apoc.create.uuid,apoc.do.case,apoc.do.when,apoc.load.json,apoc.map.flatten,apoc.map.removeKeys,apoc.merge.node,apoc.merge.relationship,apoc.nodes.get,apoc.text.regreplace,apoc.text.replace,apoc.text.split
apoc.import.file.enabled=true
apoc.import.file.use_neo4j_config=false

dbms.memory.heap.max_size=16G
```

Note: The `push` operation can require a large amount of memory when processing large datasets. Praetorian has experienced heap sizes of 10G when processing large Kubernetes clusters. Setting an appropriate `dbms.memory.heap.max_size` will keep the import process from crashing.

Finally, after setting all of the configuration options, restart the DBMS if it is currently running so all setup tasks are loaded into the running DBMS.

# Usage

## Neo4j Authentication
Konstellation requires a Neo4j database to perform the `push` and `query` functions. By default, it will look for the database at `bolt://localhost:7687` with `neo4j` as the username and password. The user may specify alternate configurations with the `--neo4juri`, `--neo4juser`, and `--neo4jpass` options.

Example:

```bash
python3 konstellation.py k8s enum --neo4juri bolt://1.2.3.4:7687 --neo4juser konstellation --neo4jpass konstellation
```


## Enumerating resources (`enum`)

The `enum` command will enumerate the specified platform with the provided credentials. Results are written to `<platform>-enum` unless an alternate directory is specified with `--enum`.

Examples:

```bash
python3 konstellation.py k8s enum
```

```
python3 konstellation.py k8s enum --enum foo/bar
```

## Loading data (`push`)
The `push` command loads the enumerated data and stores it in the Neo4j database.

Examples:

Loading data with default enum directory (`k8s-enum`).
```bash
python3 konstellation.py k8s push
```

Pushing with a custom enum directory.
```
python3 konstellation.py k8s push --enum foo/bar
```

Re-running all relationship mapping.
```
python3 konstellation.py k8s push --relationships
```

Run a single relationship mapping.
```
python3 konstellation.py k8s push --relationships --relationship-name "Cluster Role Bindings"
```


## Querying data (`query`)
Konstellation's query operation performs the specified queries on the `push`ed data. It writes the `query` results to the `<platform>-results` directory unless otherwise specified with the `--results/-r` option. By default, Konstellation runs all queries defined for a platform, but a user may perform single queries using the `--name` option.

Examples:

Run all queries for the k8s platform.
```
python3 ./konstellation.py k8s query 
```

Print results to stdout as well as write to files.
```
python3 ./konstellation.py k8s query --print
```

Run a single query.
```
python3 ./konstellation.py k8s query --name "Resources that can create or modify role bindings"
```

Write `query` results to a non-default directory.
```
python3 ./konstellation.py k8s query --results custom/dir
```

# Schema
The structured output of the source JSON drives the schema. Konstellation parses the raw json files obtained during enumeration and transforms them into nodes and relationships based on the `resources/<platform>/config.yml`. Using the data to drive the schema allows for rapid development and default handling of new data types.

### K8s/Kubernetes
Kubernetes has two notable deviations to the raw enumeration data structure in regards to RoleBindings and subresources.

RoleBindings are present; however, in addition to the `(role)-[:ROLEREF]->(rolebinding)-[:SUBJECT]->(subject)` mapping, developers implemented more concise representation. A `ROLE_BINDING` relationship between the subject and role (`(subject)-[:ROLE_BINDING]->(role)`) replaces the more verbose node structure. This approach simplifies the graph structure and complexity, and queries.

Subresources are map as relationships to the parent node where the relationship name is verb + subresource. For example, a role with the `get` verb on `pod/exec` would have the `GET_EXEC` relationship mapped to the appropriate pod: `(role)-[:GET_EXEC]->(p:Pod)`

#### Meta Resource Nodes

A special "meta" node exists for each resource type to represent the resource itself and map wildcard permissions. The nodes have the name and kind set as the resource type, and a `type` property set to `resource`. These meta-resource types are useful for finding excessive privileges. An example is below looking for non-`kube-system` namespaced service accounts that can read all secrets.

```cypher
MATCH (sa:ServiceAccount)-[:ROLE_BINDING]->(x)-[:GET]->(s:Secret {type: 'resource'}) WHERE NOT sa.namespace = 'kube-system' RETURN *
```