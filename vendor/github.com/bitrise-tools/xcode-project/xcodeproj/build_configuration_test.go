package xcodeproj

import (
	"testing"

	"github.com/bitrise-tools/xcode-project/pretty"
	"github.com/bitrise-tools/xcode-project/serialized"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func TestParseBuildConfiguration(t *testing.T) {
	var raw serialized.Object
	_, err := plist.Unmarshal([]byte(rawBuildConfiguration), &raw)
	require.NoError(t, err)

	buildConfiguration, err := parseBuildConfiguration("13E76E381F4AC90A0028096E", raw)
	require.NoError(t, err)
	// fmt.Printf("buildConfiguration:\n%s\n", )
	require.Equal(t, expectedBuildConfiguration, pretty.Object(buildConfiguration))
}

const rawBuildConfiguration = `
{
	13E76E381F4AC90A0028096E /* Debug */ = {
		isa = XCBuildConfiguration;
		buildSettings = {
			ALWAYS_SEARCH_USER_PATHS = NO;
			CLANG_ANALYZER_NONNULL = YES;
			CLANG_ANALYZER_NUMBER_OBJECT_CONVERSION = YES_AGGRESSIVE;
			CLANG_CXX_LANGUAGE_STANDARD = "gnu++14";
			CLANG_CXX_LIBRARY = "libc++";
			CLANG_ENABLE_MODULES = YES;
			CLANG_ENABLE_OBJC_ARC = YES;
			CLANG_WARN_BLOCK_CAPTURE_AUTORELEASING = YES;
			CLANG_WARN_BOOL_CONVERSION = YES;
			CLANG_WARN_COMMA = YES;
			CLANG_WARN_CONSTANT_CONVERSION = YES;
			CLANG_WARN_DIRECT_OBJC_ISA_USAGE = YES_ERROR;
			CLANG_WARN_DOCUMENTATION_COMMENTS = YES;
			CLANG_WARN_EMPTY_BODY = YES;
			CLANG_WARN_ENUM_CONVERSION = YES;
			CLANG_WARN_INFINITE_RECURSION = YES;
			CLANG_WARN_INT_CONVERSION = YES;
			CLANG_WARN_NON_LITERAL_NULL_CONVERSION = YES;
			CLANG_WARN_OBJC_LITERAL_CONVERSION = YES;
			CLANG_WARN_OBJC_ROOT_CLASS = YES_ERROR;
			CLANG_WARN_RANGE_LOOP_ANALYSIS = YES;
			CLANG_WARN_STRICT_PROTOTYPES = YES;
			CLANG_WARN_SUSPICIOUS_MOVE = YES;
			CLANG_WARN_UNGUARDED_AVAILABILITY = YES_AGGRESSIVE;
			CLANG_WARN_UNREACHABLE_CODE = YES;
			CLANG_WARN__DUPLICATE_METHOD_MATCH = YES;
			CODE_SIGN_IDENTITY = "iPhone Developer";
			COPY_PHASE_STRIP = NO;
			DEBUG_INFORMATION_FORMAT = dwarf;
			ENABLE_STRICT_OBJC_MSGSEND = YES;
			ENABLE_TESTABILITY = YES;
			GCC_C_LANGUAGE_STANDARD = gnu11;
			GCC_DYNAMIC_NO_PIC = NO;
			GCC_NO_COMMON_BLOCKS = YES;
			GCC_OPTIMIZATION_LEVEL = 0;
			GCC_PREPROCESSOR_DEFINITIONS = (
				"DEBUG=1",
				"$(inherited)",
			);
			GCC_WARN_64_TO_32_BIT_CONVERSION = YES;
			GCC_WARN_ABOUT_RETURN_TYPE = YES_ERROR;
			GCC_WARN_UNDECLARED_SELECTOR = YES;
			GCC_WARN_UNINITIALIZED_AUTOS = YES_AGGRESSIVE;
			GCC_WARN_UNUSED_FUNCTION = YES;
			GCC_WARN_UNUSED_VARIABLE = YES;
			IPHONEOS_DEPLOYMENT_TARGET = 11.0;
			MTL_ENABLE_DEBUG_INFO = YES;
			ONLY_ACTIVE_ARCH = YES;
			SDKROOT = iphoneos;
		};
		name = Debug;
	};
}`

const expectedBuildConfiguration = `{
	"ID": "13E76E381F4AC90A0028096E",
	"Name": "Debug",
	"BuildSettings": {
		"ALWAYS_SEARCH_USER_PATHS": "NO",
		"CLANG_ANALYZER_NONNULL": "YES",
		"CLANG_ANALYZER_NUMBER_OBJECT_CONVERSION": "YES_AGGRESSIVE",
		"CLANG_CXX_LANGUAGE_STANDARD": "gnu++14",
		"CLANG_CXX_LIBRARY": "libc++",
		"CLANG_ENABLE_MODULES": "YES",
		"CLANG_ENABLE_OBJC_ARC": "YES",
		"CLANG_WARN_BLOCK_CAPTURE_AUTORELEASING": "YES",
		"CLANG_WARN_BOOL_CONVERSION": "YES",
		"CLANG_WARN_COMMA": "YES",
		"CLANG_WARN_CONSTANT_CONVERSION": "YES",
		"CLANG_WARN_DIRECT_OBJC_ISA_USAGE": "YES_ERROR",
		"CLANG_WARN_DOCUMENTATION_COMMENTS": "YES",
		"CLANG_WARN_EMPTY_BODY": "YES",
		"CLANG_WARN_ENUM_CONVERSION": "YES",
		"CLANG_WARN_INFINITE_RECURSION": "YES",
		"CLANG_WARN_INT_CONVERSION": "YES",
		"CLANG_WARN_NON_LITERAL_NULL_CONVERSION": "YES",
		"CLANG_WARN_OBJC_LITERAL_CONVERSION": "YES",
		"CLANG_WARN_OBJC_ROOT_CLASS": "YES_ERROR",
		"CLANG_WARN_RANGE_LOOP_ANALYSIS": "YES",
		"CLANG_WARN_STRICT_PROTOTYPES": "YES",
		"CLANG_WARN_SUSPICIOUS_MOVE": "YES",
		"CLANG_WARN_UNGUARDED_AVAILABILITY": "YES_AGGRESSIVE",
		"CLANG_WARN_UNREACHABLE_CODE": "YES",
		"CLANG_WARN__DUPLICATE_METHOD_MATCH": "YES",
		"CODE_SIGN_IDENTITY": "iPhone Developer",
		"COPY_PHASE_STRIP": "NO",
		"DEBUG_INFORMATION_FORMAT": "dwarf",
		"ENABLE_STRICT_OBJC_MSGSEND": "YES",
		"ENABLE_TESTABILITY": "YES",
		"GCC_C_LANGUAGE_STANDARD": "gnu11",
		"GCC_DYNAMIC_NO_PIC": "NO",
		"GCC_NO_COMMON_BLOCKS": "YES",
		"GCC_OPTIMIZATION_LEVEL": "0",
		"GCC_PREPROCESSOR_DEFINITIONS": [
			"DEBUG=1",
			"$(inherited)"
		],
		"GCC_WARN_64_TO_32_BIT_CONVERSION": "YES",
		"GCC_WARN_ABOUT_RETURN_TYPE": "YES_ERROR",
		"GCC_WARN_UNDECLARED_SELECTOR": "YES",
		"GCC_WARN_UNINITIALIZED_AUTOS": "YES_AGGRESSIVE",
		"GCC_WARN_UNUSED_FUNCTION": "YES",
		"GCC_WARN_UNUSED_VARIABLE": "YES",
		"IPHONEOS_DEPLOYMENT_TARGET": "11.0",
		"MTL_ENABLE_DEBUG_INFO": "YES",
		"ONLY_ACTIVE_ARCH": "YES",
		"SDKROOT": "iphoneos"
	}
}`
