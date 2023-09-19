module github.com/micromdm/nanocmd

go 1.19

require (
	github.com/alexedwards/flow v0.0.0-20220806114457-cf11be9e0e03
	github.com/go-sql-driver/mysql v1.7.1
	github.com/google/uuid v1.3.1
	github.com/groob/plist v0.0.0-20220217120414-63fa881b19a5
	github.com/jessepeterson/mdmcommands v0.0.0-20230517161100-c5ca4128e1e3
	github.com/peterbourgon/diskv/v3 v3.0.1
	go.mozilla.org/pkcs7 v0.0.0-00010101000000-000000000000
)

require github.com/google/btree v1.1.2 // indirect

replace go.mozilla.org/pkcs7 => github.com/smallstep/pkcs7 v0.0.0-20230302202335-4c094085c948
