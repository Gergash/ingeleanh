def reciprocal_rank_fusion(
    dense_results: list[dict],
    sparse_results: list[dict],
    k: int = 60,
) -> list[dict]:
    """Combina rankings densos y dispersos con RRF (Cormack et al., 2009)."""
    scores: dict[str, float] = {}
    chunks: dict[str, dict] = {}

    for rank, chunk in enumerate(dense_results, start=1):
        chunk_id = str(chunk["id"])
        scores[chunk_id] = scores.get(chunk_id, 0) + 1.0 / (k + rank)
        chunks[chunk_id] = chunk

    for rank, chunk in enumerate(sparse_results, start=1):
        chunk_id = str(chunk["id"])
        scores[chunk_id] = scores.get(chunk_id, 0) + 1.0 / (k + rank)
        chunks[chunk_id] = chunk

    sorted_ids = sorted(scores.keys(), key=lambda x: scores[x], reverse=True)
    return [{**chunks[cid], "rrf_score": scores[cid]} for cid in sorted_ids]
