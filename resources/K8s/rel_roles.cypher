//Roles
CALL apoc.load.json('{{ path }}', '$.items[*]') YIELD value as items
WITH items, items.kind as kind, items.metadata as m, items.metadata.namespace as namespace

// Get the role node
MATCH (role {name: m.name, uid: m.uid})
WITH role, items, kind, namespace

// Iterate over each rule
UNWIND items.rules as rules
    //
    WITH rules.resources as resources, rules.verbs as verbs, apoc.convert.toList(rules.resourceNames) as resourceNames, namespace, kind, role, rules.apiGroups as groups
    UNWIND resources as resource
        UNWIND verbs as verb
            UNWIND groups as group
                // resource - lowercase, strip the trailing s, and replace the wildcard * with .* to convert it to a regex
                // and split the resource by / to get the subresource
                WITH *, toLower(apoc.text.replace(apoc.text.replace(apoc.text.replace(resource, '/.*$', ''), '\*', '.*'), 's$', '')) as resource, apoc.text.split(resource, '/')[1] as subresource 
                // convert group wildcard to regex wildcard
                WITH *, apoc.text.replace(group, '\*', '.*') as group
                    // If a subresource value exists set the relationship name to VERB_SUBRESOURCE
                    // otherwise set the verb as the relationship name
                    // In both cases, a wildcard verb is replaced by FULL_CONTROL
                    CALL apoc.do.when(
                        subresource IS NOT NULL,
                        "RETURN toUpper(apoc.text.replace(verb, '\*', 'FULL_CONTROL') + '_' + subresource) as relname",
                        "RETURN toUpper(apoc.text.replace(verb, '\*', 'FULL_CONTROL')) as relname",
                        {subresource: subresource, verb: verb}
                    ) YIELD value as relname

                    WITH role, resource, relname.relname as relname, resourceNames, group, namespace
                        // When resource names have been specified, add them to the WHERE clause
                        // otherwise regex match only on the resource
                        CALL apoc.do.case(
                            [
                                // API group and resource names present
                                // Seems as though the apiGroups key is always present and it produces a list with an empty string, but
                                // resourceNames is only present when there are values. So an empty list and empty string check for the
                                // first element is requied groups and resourceNames has to be compared to null.
                                //
                                // resource is always treated as a regex
                                NOT isEmpty(group) AND NOT group = "" and resourceNames IS NOT NULL,
                                'MATCH (res {namespace: namespace}) WHERE res.kind =~ resource and res.name in resourceNames and res.`spec.group` =~ group RETURN res',

                                // API group present
                                NOT isEmpty(group) AND NOT group = "",
                                'MATCH (res {namespace: namespace}) WHERE res.kind =~ resource and res.`spec.group` =~ group RETURN res',

                                // resourceNames present
                                resourceNames IS NOT NULL,
                                'MATCH (res {namespace: namespace}) WHERE res.kind =~ resource and res.name in resourceNames RETURN res'
                            ],
                            
                            // default
                            'MATCH (res {namespace: namespace}) WHERE res.kind =~ resource RETURN res',
                            {resource: resource, resourceNames: resourceNames, group: group, namespace: namespace}
                        ) YIELD value as res

                        WITH res, role, relname
                            UNWIND res as resNode
                                // merge the relationship between the role and matched resource with the relname
                                CALL apoc.merge.relationship(role, relname, {}, {}, resNode.res, {}) YIELD rel as rel
                                RETURN rel