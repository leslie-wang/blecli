import uasyncio as asyncio
import aioble
import bluetooth
import time

ble_apprearance = 0x0300
ble_advertiseing_interval = 2000

# BLE UUIDs
SERVICE_UUID = bluetooth.UUID(0x181A)     # Environmental Sensing
WRITE_CHAR_UUID = bluetooth.UUID(0x2A6E)   # Temperature (write)
NOTIFY_CHAR_UUID = bluetooth.UUID(0x2A1C)  # Notify

# Create BLE service and characteristics
service = aioble.Service(SERVICE_UUID)

write_char = aioble.Characteristic(
    service,
    WRITE_CHAR_UUID,
    read=True,
    write=True,
    write_no_response=True,
)

notify_char = aioble.Characteristic(
    service,
    NOTIFY_CHAR_UUID,
    read=True,
    notify=True,
)

# Register service
aioble.register_services(service)

# Main advertising and connection loop
async def connection_handler():
    while True:
        print("Advertising...")
        try:
            async with await aioble.advertise(
                ble_advertiseing_interval,
                name="pico2w_ble",
                services=[SERVICE_UUID],
                appearance=ble_apprearance
            ) as conn:
                print("Connected to:", conn.device)

                while True:
                    try:
                        writer = await write_char.written()
                        data1 = bytes(writer.data) if hasattr(writer, "data") else None
                        print("Write received:", data1)
                    except Exception as e:
                        print("write_char.written() failed:", e)
                        break

                    if not conn or not conn.is_connected():
                        print("Connection lost before write.")
                        break

                    print("Written by device")
                    data = write_char.read()
                    #data = "123"
                    if data:
                        print("Write received from client:", data)

                        try:
                            print("Send notify:", conn.device)
                            notify_char.notify(conn, b"ACK: hi")
                            print("Notify sent successfully")
                            
                            time.sleep(3)
                        except Exception as e:
                            print("Error sending notification:", e)
                            #break
        except Exception as e:
            print("Advertising or connection failed:", e)

        print("Client disconnected, restarting advertise loop.")

# Run the BLE handler
asyncio.run(connection_handler())
