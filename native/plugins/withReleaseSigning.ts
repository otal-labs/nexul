import { type ConfigPlugin, withAppBuildGradle } from "expo/config-plugins";

// Gradle reads these from -P flags or ORG_GRADLE_PROJECT_* environment variables; without them release stays debug-signed.
const signing = `
if (project.hasProperty("nexulKeystoreFile")) {
    android {
        signingConfigs {
            release {
                storeFile file(nexulKeystoreFile)
                storePassword nexulKeystorePassword
                keyAlias nexulKeyAlias
                keyPassword nexulKeyPassword
            }
        }
        buildTypes {
            release {
                signingConfig signingConfigs.release
            }
        }
    }
}
`;

const withReleaseSigning: ConfigPlugin = (config) =>
  withAppBuildGradle(config, (gradle) => {
    if (gradle.modResults.contents.includes("nexulKeystoreFile")) {
      return gradle;
    }
    gradle.modResults.contents += signing;
    return gradle;
  });

export default withReleaseSigning;
