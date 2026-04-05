import { Text, YStack, Button } from 'tamagui';

export default function ScanScreen() {
  return (
    <YStack flex={1} backgroundColor="$background" alignItems="center" justifyContent="center" padding="$4">
      <Text fontSize="$6" fontWeight="bold" marginBottom="$4">
        Scan Visitor QR
      </Text>
      {/* TODO: integrate expo-camera CameraView with QR barcode scanning */}
      <YStack
        width="80%"
        aspectRatio={1}
        backgroundColor="$color3"
        borderRadius="$4"
        alignItems="center"
        justifyContent="center"
        marginBottom="$6"
      >
        <Text color="$color10">Camera preview</Text>
      </YStack>
      <Text color="$color10" marginBottom="$4">or enter OTP manually</Text>
      <Button theme="active" width="80%">
        Enter OTP
      </Button>
    </YStack>
  );
}
