// Read-only: only listCollections, find, aggregate and countDocuments run here. Never add a write.
// The database comes from the connection string, so a typo'd name fails the preflight below instead of exporting zero Streams.
// Stdout is exactly one JSON document, so redirect it straight to a file.
(function () {
  const STREAMS = "formata_builder_streams";
  const PROCESSES = "processes";

  const names = db.getCollectionNames();
  for (const required of [STREAMS, PROCESSES]) {
    if (!names.includes(required)) {
      throw new Error(
        "Collection " + required + " not found in database '" + db.getName() + "'. Existing: " + names.join(", ")
      );
    }
  }

  const hex = (id) => (id && typeof id.toHexString === "function" ? id.toHexString() : String(id));

  const streams = db
    .getCollection(STREAMS)
    .find({})
    .sort({ _id: 1 })
    .toArray()
    .map((doc) => {
      const { _id, stream, ...fields } = doc;
      return { _id: hex(_id), stream, fields: JSON.parse(JSON.stringify(fields)) };
    });

  // A missing `workflowKey` groups as null; server code treats those as the on-disk "workflow" key.
  // `dpp` is a subdocument written only when a DPP is issued (Process.DPP, bson "dpp,omitempty").
  const processGroups = db
    .getCollection(PROCESSES)
    .aggregate(
      [
        {
          $group: {
            _id: { workflowKey: "$workflowKey", status: "$status" },
            count: { $sum: 1 },
            dppCount: { $sum: { $cond: [{ $eq: [{ $type: "$dpp" }, "object"] }, 1, 0] } },
          },
        },
        { $sort: { "_id.workflowKey": 1, "_id.status": 1 } },
      ],
      { allowDiskUse: true }
    )
    .toArray()
    .map((row) => ({
      workflowKey: row._id.workflowKey === undefined ? null : row._id.workflowKey,
      status: row._id.status === undefined ? null : row._id.status,
      count: row.count,
      dppCount: row.dppCount,
    }));

  const processesTotal = db.getCollection(PROCESSES).countDocuments({});
  const processesWithDpp = db.getCollection(PROCESSES).countDocuments({ dpp: { $type: "object" } });

  const out = {
    schema: 1,
    exportedAt: new Date().toISOString(),
    database: db.getName(),
    serverVersion: db.version(),
    counts: {
      streamDocuments: db.getCollection(STREAMS).countDocuments({}),
      streamsExported: streams.length,
      processesTotal,
      processesWithDpp,
    },
    streams,
    processGroups,
  };

  print(JSON.stringify(out));
})();
