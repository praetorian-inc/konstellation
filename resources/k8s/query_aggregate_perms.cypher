MATCH (n)-[r1]->(m)
WHERE r1.aggregated = "true"
AND NOT EXISTS {
  MATCH (n)-[r2]->(m)
  WHERE type(r1) = type(r2) AND r2.aggregated IS NULL
}
RETURN n, r1, m