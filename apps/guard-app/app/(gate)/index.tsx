import { Text, YStack, XStack, Button, ScrollView } from 'tamagui';

export default function EntryLogScreen() {
  return (
    <YStack flex={1} backgroundColor="$background" padding="$4">
      <Text fontSize="$6" fontWeight="bold" marginBottom="$4">
        Today's Entry Log
      </Text>
      <ScrollView>
        {/* TODO: render list of VisitorEntry records for today */}
        <YStack alignItems="center" marginTop="$8">
          <Text color="$color10">No entries yet today</Text>
        </YStack>
      </ScrollView>
      <XStack gap="$3" marginTop="$4">
        <Button flex={1} theme="active" onPress={() => {}}>
          Log Walk-in
        </Button>
        <Button flex={1} theme="active" onPress={() => {}}>
          Log Delivery
        </Button>
      </XStack>
    </YStack>
  );
}
