CALL apoc.load.json('{{ path }}', '{{jsonPath}}') YIELD value as pods

// create the generic pod resource
WITH pods
CALL apoc.merge.node(['Pod'], {kind: 'pod', name: 'pod', type: 'resource', `spec.group`: apoc.text.split(pods.apiVersion, "/")[0]}) YIELD node

WITH pods
UNWIND pods as pod
    MERGE (p:Pod {name: pod.metadata.name, uid: pod.metadata.uid, namespace: pod.metadata.namespace})
    WITH p ,pod, apoc.map.flatten(pod) as flat
    WITH p, pod, flat, keys(flat) as keys
    CALL apoc.create.setProperties(p,[k in keys |k], [k in keys | apoc.text.regreplace(apoc.text.regreplace(apoc.convert.toJson(flat[k]), '^"', ''), '"$', '')]) YIELD node as n
    WITH *
        SET p.kind = 'pod' //gets overwritten in the apoc call above

    WITH pod, p
    UNWIND pod.spec as spec
        UNWIND spec.containers as container
        WITH container, p, pod
        CALL apoc.merge.node(['Container'], {name: container.name, id: apoc.create.uuid(), kind: 'container'}, {namespace: pod.metadata.namespace}) YIELD node as c
        WITH c ,container, apoc.map.flatten(container) as flat, p
        WITH c, container, flat, keys(flat) as keys, p
        CALL apoc.create.setProperties(c,[k in keys |k], [k in keys | apoc.text.regreplace(apoc.text.regreplace(apoc.convert.toJson(flat[k]), '^"', ''), '"$', '')]) YIELD node as n
        WITH p, n
        MERGE (p)-[:POD]->(n)
        RETURN *
