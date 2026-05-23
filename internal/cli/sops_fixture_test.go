package cli

import (
	"os"
	"path/filepath"
	"testing"
)

const cliTestAgeKey = `# created: 2026-05-23T08:04:13+01:00
# public key: age1xlp99sh4c9dheuskt900y83mdcxdhjp0zhejf2l4lkh6fdzl0dsqsx4yrx
AGE-SECRET-KEY-1640MNDQ274MATFJR6D6R6GFW3FUDJLDUAM84TXRM9L70FUCEXV0QMEH0CE
`

const cliTestEncryptedYAML = `version: 1
kind: env
name: ENC[AES256_GCM,data:Riq9aP5zSklrQVihLs8Z6cyGFoTx9m8=,iv:afiJHqzI77QqsAAPUTkTo2x9NFHIAtVwUhZqso2SUHY=,tag:Tr1eyGNfEykmeP0NsEEeTQ==,type:str]
values:
    APP_ENV: ENC[AES256_GCM,data:EkOfgsyFrpZH4g==,iv:avltATTRTS4vEsXi1IISZD0zo/4+M+Wx/3hOL3tbj+I=,tag:uzT57+ASpUVueI0RjudbPA==,type:str]
    DB_HOST: ENC[AES256_GCM,data:CyE=,iv:wUXVOYuWG9bcInf5CS3B7nUZMYXwYxveTWs5bbZdB6c=,tag:2LNNfJcBu5MFFyzot71UiA==,type:str]
sops:
    age:
        - recipient: age1xlp99sh4c9dheuskt900y83mdcxdhjp0zhejf2l4lkh6fdzl0dsqsx4yrx
          enc: |
            -----BEGIN AGE ENCRYPTED FILE-----
            YWdlLWVuY3J5cHRpb24ub3JnL3YxCi0+IFgyNTUxOSByUmNYSUREdEhrSlZ6a2pt
            SUtKZjMrOHNCc2l6TEdLRTErbWRTK0hTWkcwCjVuMmJucC93WWQzVnBoMllnWGdq
            ZGNPS3dEbThVMGtpc2VPQXo3cDZaTE0KLS0tIDZEeU1hK3d5UVZDWllBZXlnU2x5
            R2orSW1LR3pyMllDZVJYR1lEazVMbzgKk6v8JQFsBbZZ7H2Jrm3wW7Stz/fc4Xws
            eDgyjtDkBhCCLjYpMh7MZoS9ryZ0YQWXquEhJIR4fZtqmCrNf5T3Xw==
            -----END AGE ENCRYPTED FILE-----
    lastmodified: "2026-05-23T07:04:13Z"
    mac: ENC[AES256_GCM,data:wbWeTqecwI4WPC0QEuJv+4POBdepZoUwnsx7qF7RLpGspMrqXAqNzMF3IVd0HVIeupWHDi42s1t6MNfTE/5adIyprH3+3Vrc900hmRGFzKERMIwATUhqgtKkhO50YQ7CGX5blevr21Jadce56Q+cXJs+rv/criVcf5V0zrhEhDo=,iv:7xhnkpYcJgtnz1jkhHW/tuZ7F4xfSJfb9/MBrW7XFGs=,tag:LZW2wKKLgMXuftGAlqiHtQ==,type:str]
    encrypted_regex: ^(name|values)$
    version: 3.12.2
`

func configureLibraryDecryptFixture(t *testing.T, root string) string {
	t.Helper()
	keyPath := filepath.Join(root, "test-age.key")
	if err := os.WriteFile(keyPath, []byte(cliTestAgeKey), 0o600); err != nil {
		t.Fatalf("write age key: %v", err)
	}
	t.Setenv("SOPS_AGE_KEY_FILE", keyPath)
	return keyPath
}
