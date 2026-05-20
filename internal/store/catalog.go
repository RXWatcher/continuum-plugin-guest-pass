package store

import "context"

// PlayableFileIDs resolves a list of host content IDs (media item IDs or
// episode IDs) to the preferred playable media-file ID for each. Picks
// the highest-resolution non-missing file available.
//
// This is the only place in the plugin that reaches into the host's
// schema. The host SDK does not yet expose a "resolve playable file"
// call, so the cross-schema reach is irreducible until that lands —
// document the public.media_files SELECT grant in the README.
func (s *Store) PlayableFileIDs(ctx context.Context, contentIDs []string) (map[string]int, error) {
	if len(contentIDs) == 0 {
		return map[string]int{}, nil
	}
	rows, err := s.pool.Query(ctx, `
WITH requested(content_id) AS (
    SELECT DISTINCT unnest($1::text[])
),
ranked AS (
    SELECT
        r.content_id,
        mf.id AS file_id,
        row_number() OVER (
            PARTITION BY r.content_id
            ORDER BY
                CASE lower(COALESCE(mf.resolution, ''))
                    WHEN '2160p' THEN 1
                    WHEN '4k'    THEN 1
                    WHEN '1080p' THEN 2
                    WHEN '720p'  THEN 3
                    WHEN '480p'  THEN 4
                    ELSE 5
                END,
                mf.id
        ) AS rn
    FROM requested r
    JOIN public.media_files mf ON (mf.content_id = r.content_id OR mf.episode_id = r.content_id)
    WHERE mf.missing_since IS NULL
)
SELECT content_id, file_id FROM ranked WHERE rn = 1`, contentIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int, len(contentIDs))
	for rows.Next() {
		var contentID string
		var fileID int
		if err := rows.Scan(&contentID, &fileID); err != nil {
			return nil, err
		}
		out[contentID] = fileID
	}
	return out, rows.Err()
}
