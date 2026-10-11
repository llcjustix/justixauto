# P12 source-release handoff correction

P12a1 continuation reported DONE and source released before its final message.
Root granted P12a2 on that explicit release. The continuation subsequently
reported one final in-scope `TemplateDetail` service/test change and its passing
affected selector. This creates a possible overlap in source-writer timing;
the work must not be described as strictly serialized at that boundary.

The added service method is required by the already approved template detail
endpoint. Its reported service/test paths are distinct from P12a2's HTTP/app
paths. Preserve the authorized change. Root told the continuation to make no
further source edits and supplied the final method/result to the API worker.
The continuation has now ended; P12a2 alone holds the writer lease. No extra
verification or rollback is authorized solely by this handoff note.

Future release messages must be final; remaining edits require a new grant.
