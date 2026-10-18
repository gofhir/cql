library ReviewProbe version '1.0'
define WithAggregate: ({1,2,3}) X with ({2}) Y such that X = Y aggregate A starting 0: A + X
define WithoutAggregate: ({1,2,3}) X without ({2}) Y such that X = Y aggregate A starting 0: A + X
define DistinctCollision: ({'nil', null}) X aggregate distinct A starting 0: A + 1
define UnionDuplicates: ({1,1,2}) union ({2})
define ExceptDuplicates: ({1,1,2}) except ({3})
