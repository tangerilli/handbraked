package common

import ("path/filepath")

func FindFileTypes(path string, extensions []string) []string {
    var results []string
    for _, pattern := range extensions {
        r, err := filepath.Glob(filepath.Join(path, "*." + pattern))
        if err != nil {
            continue
        }
        results = append(results, r...)
    }
    return results
}