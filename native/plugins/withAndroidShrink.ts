import { type ConfigPlugin, withAppBuildGradle, withGradleProperties } from "expo/config-plugins";

// R8 drops the Java and Kotlin code nothing calls and the resources nothing reads; the template leaves both off.
const shrink = { "android.enableMinifyInReleaseBuilds": "true", "android.enableShrinkResourcesInReleaseBuilds": "true" };

// expo-updates' signature check brings all of BouncyCastle; R8 drops the unused post-quantum engines, but not the
// constant tables they load as resources, which nothing else reads.
const excludes = `
android {
    packagingOptions {
        resources {
            excludes += ["org/bouncycastle/pqc/**"]
        }
    }
}
`;

const withAndroidShrink: ConfigPlugin = (config) => {
  const withProperties = withGradleProperties(config, (gradle) => {
    for (const [key, value] of Object.entries(shrink)) {
      gradle.modResults = gradle.modResults.filter((item) => item.type !== "property" || item.key !== key);
      gradle.modResults.push({ type: "property", key, value });
    }
    return gradle;
  });
  return withAppBuildGradle(withProperties, (gradle) => {
    if (gradle.modResults.contents.includes("org/bouncycastle/pqc/**")) {
      return gradle;
    }
    gradle.modResults.contents += excludes;
    return gradle;
  });
};

export default withAndroidShrink;
