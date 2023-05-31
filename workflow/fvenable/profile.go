package fvenable

const ProfileTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>EncryptCertPayloadUUID</key>
			<string>00FD73C8-9AE6-4A1E-BBD0-22315CDE7533</string>
			<key>Location</key>
			<string>MDM server</string>
			<key>PayloadIdentifier</key>
			<string>io.micromdm.wf.fvenable.v1.payload.escrow</string>
			<key>PayloadType</key>
			<string>com.apple.security.FDERecoveryKeyEscrow</string>
			<key>PayloadUUID</key>
			<string>5943E786-24DD-4DE7-A27C-3F84B55A7A4B</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>Defer</key>
			<true/>
			<key>DeferForceAtUserLoginMaxBypassAttempts</key>
			<integer>0</integer>
			<key>Enable</key>
			<string>On</string>
			<key>PayloadIdentifier</key>
			<string>io.micromdm.wf.fvenable.v1.payload.filevault</string>
			<key>PayloadType</key>
			<string>com.apple.MCX.FileVault2</string>
			<key>PayloadUUID</key>
			<string>ED92D4D5-FF80-430B-AACC-67C32B685E59</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ShowRecoveryKey</key>
			<false/>
		</dict>
		<dict>
			<key>PayloadContent</key>
			<data>__CERTIFICATE__</data>
			<key>PayloadIdentifier</key>
			<string>io.micromdm.wf.fvenable.v1.payload.pkcs1</string>
			<key>PayloadType</key>
			<string>com.apple.security.pkcs1</string>
			<key>PayloadUUID</key>
			<string>00FD73C8-9AE6-4A1E-BBD0-22315CDE7533</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>FileVault</string>
	<key>PayloadIdentifier</key>
	<string>io.micromdm.wf.fvenable.v1.profile</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>49DDB449-163E-4408-B05D-FA4814CDEE3E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>
`
