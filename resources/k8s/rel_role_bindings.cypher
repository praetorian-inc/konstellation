CALL apoc.load.json('{{ path }}', '$.items[*]') YIELD value as rbs
WITH rbs.metadata.name as rbname, rbs.kind as rbkind, rbs.roleRef.name as rname, rbs.roleRef.kind as rkind, rbs.subjects as subjects, rbs.namespace as namespace
    UNWIND subjects as s
        WITH *, s.kind as kind, apoc.text.split(s.name, ":") as user

        // If the node doesn't exist, create it with a a `defined: false` property
        OPTIONAL MATCH (sub {name: s.name, kind: toLower(s.kind), `spec.group`: s.apiGroup})
        WITH *
        CALL apoc.do.case([
            sub IS NULL,
            'CALL apoc.merge.node([s.kind], {name: s.name, kind: toLower(s.kind)}, {`spec.group`: s.apiGroup, defined: false}) YIELD node RETURN node'
            ],
            'RETURN NULL',
            {s:s}
        ) YIELD value

        WITH *
        CALL apoc.do.case(
        [
            // system:serviceaccount:foo:admin-deployer
            kind = "User" AND user[3] IS NOT NULL,
            'MATCH (subject:ServiceAccount {name: user[3], namespace: user[2]}) RETURN subject',

            // system:kube-proxy
            kind = "User" and size(user) = 2,
            'MATCH (subject:ServiceAccount {name: user[1]}) RETURN subject'
        ],
            'MATCH (subject {name: name}) WHERE subject.kind = toLower(kind) return subject',
            {kind: toLower(s.kind), name: s.name, user: apoc.text.split(s.name, ":"), apiGroup: s.apiGroup}
        ) YIELD value as subject
        WITH subject.subject as subject, rkind, rname, rbname, rbkind, namespace
        
        // get role
        WITH *
        CALL apoc.merge.node([rkind], {name: rname}) YIELD node as role
        
        // create role binding relationship between subject and role
        WITH *
        CALL apoc.do.case(
        [
            // roles have a namespace
            toLower(role.kind) = "role",
            "MERGE (subject)-[r:ROLE_BINDING {name: rbname, kind: rbkind, namespace: role.namespace}]->(role)"
        ],
            // clusterroles do not have namespace
            "MERGE (subject)-[r:ROLE_BINDING {name: rbname, kind: rbkind}]->(role)",
            {subject: subject, rbname: rbname, rbkind: rbkind, role: role}
        ) YIELD value
        //MERGE (subject)-[r:ROLE_BINDING {name: rbname, kind: rbkind, namespace: namespace}]->(role)


        // link the binding
        WITH role, subject, rbname, rbkind
        MATCH (binding {name: rbname, kind: rbkind})

        WITH *
        MERGE (role)<-[:ROLE]-(binding)-[:SUBJECT]->(subject)