package main

import "testing"

func TestParseMongoDBStorageOptions(t *testing.T) {
	tests := []struct {
		name             string
		options          string
		wantDatabase     string
		wantPrefix       string
		wantErr          bool
		wantCollection   string
		sourceCollection string
	}{
		{
			name:             "empty",
			sourceCollection: "subsystem_profiles",
			wantCollection:   "subsystem_profiles",
		},
		{
			name:             "database",
			options:          "database=nanocmd_prod",
			wantDatabase:     "nanocmd_prod",
			sourceCollection: "subsystem_profiles",
			wantCollection:   "subsystem_profiles",
		},
		{
			name:             "database alias and collection prefix",
			options:          "db=nanocmd_prod,collection_prefix=team_a",
			wantDatabase:     "nanocmd_prod",
			wantPrefix:       "team_a",
			sourceCollection: "subsystem_profiles",
			wantCollection:   "team_a_subsystem_profiles",
		},
		{
			name:             "collection prefix with trailing underscore",
			options:          "collection_prefix=team_a_",
			wantPrefix:       "team_a_",
			sourceCollection: "engine",
			wantCollection:   "team_a_engine",
		},
		{
			name:    "invalid",
			options: "collection_prefix",
			wantErr: true,
		},
		{
			name:    "unknown",
			options: "collection=profiles",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := parseMongoDBStorageOptions(tt.options)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}

			if have, want := opts.database, tt.wantDatabase; have != want {
				t.Errorf("database: have %q, want %q", have, want)
			}
			if have, want := opts.collectionPrefix, tt.wantPrefix; have != want {
				t.Errorf("collectionPrefix: have %q, want %q", have, want)
			}
			if have, want := mongoDBCollectionName(opts.collectionPrefix, tt.sourceCollection), tt.wantCollection; have != want {
				t.Errorf("collection: have %q, want %q", have, want)
			}
		})
	}
}

func TestParseStorageRejectsMongoAlias(t *testing.T) {
	_, err := parseStorage("mongo", "", "")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBDatabaseFromURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		want    string
		wantErr bool
	}{
		{name: "empty"},
		{name: "no database", uri: "mongodb://localhost:27017"},
		{name: "database", uri: "mongodb://localhost:27017/nanocmd", want: "nanocmd"},
		{name: "database with options", uri: "mongodb://localhost:27017/nanocmd?retryWrites=true", want: "nanocmd"},
		{name: "srv database", uri: "mongodb+srv://example.mongodb.net/nanocmd", want: "nanocmd"},
		{name: "escaped database", uri: "mongodb://localhost:27017/nano%2Dcmd", want: "nano-cmd"},
		{name: "invalid escape", uri: "mongodb://localhost:27017/%zz", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			have, err := mongoDBDatabaseFromURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if have != tt.want {
				t.Errorf("have %q, want %q", have, tt.want)
			}
		})
	}
}
