package utils

func JsonMergePatch(base map[string]any, patch map[string]any) map[string]any {
	for k, v := range patch {
		if v == nil {
			delete(base, k)
			continue
		}

		if bv, ok := base[k].(map[string]any); ok {
			if pv, ok := v.(map[string]any); ok {
				base[k] = JsonMergePatch(bv, pv)
				continue
			}
		}

		base[k] = v
	}
	return base
}
