CALL apoc.load.json('{{ path }}', '$.items[*]') YIELD value as items

WITH items, items.kind as kind, items.metadata as m, apoc.text.split(items.apiVersion, "/")[0] as specGroup
// create a resource node that will catch wildcard permissions
CALL apoc.merge.node([kind], {kind: toLower(kind), name: toLower(kind), type: 'resource', `spec.group`: specGroup}) YIELD node

WITH items, kind, m, specGroup
//build node dynamically
CALL apoc.merge.node([kind], {kind: toLower(kind), name: m.name, uid: m.uid, `spec.group`: specGroup}) YIELD node as n

//set metadata props
WITH items, n, m, keys(m) as keys, kind
// kind of ugly, but we're converting the values to json, so they can be converted back to a map later. 
// However, it wraps bare values in double quotes, so two apoc.text.regreplace calls exist to remove those
// since backreferences aren't supported.
CALL apoc.create.setProperties(n,[k in keys |k], [k in keys | apoc.text.regreplace(apoc.text.regreplace(apoc.convert.toJson(m[k]), '^"', ''), '"$', '')]) YIELD node as updated

// flatten and save props
WITH updated, apoc.map.removeKeys(apoc.map.flatten(items), ["metadata"]) as flat, kind
WITH updated, keys(flat) as keys, flat, kind
CALL apoc.create.setProperties(updated,[k in keys |k], [k in keys | apoc.text.regreplace(apoc.text.regreplace(apoc.convert.toJson(flat[k]), '^"', ''), '"$', '')]) YIELD node as n2
WITH n2, kind
CALL apoc.create.setProperty(n2, "kind", toLower(kind)) YIELD node as n3
RETURN n3
