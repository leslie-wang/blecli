import uasyncio as asyncio
import aioble
import bluetooth

aioble.log_level=2
ble_apprearance = 0x0300
ble_advertiseing_interval = 2000

# BLE UUIDs
SERVICE_UUID = bluetooth.UUID(0x181A)     # Environmental Sensing
WRITE_CHAR_UUID = bluetooth.UUID(0x6e40)   # Temperature (write)
NOTIFY_CHAR_UUID = bluetooth.UUID(0x6e41)  # Notify

# Create BLE service and characteristics
'''
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
'''
'''
def run_service():
    # Create BLE service and characteristics
    service = aioble.Service(SERVICE_UUID)

    global write_char = aioble.Characteristic(
        service,
        WRITE_CHAR_UUID,
        read=True,
        write=True,
        write_no_response=True,
    )

    global notify_char = aioble.Characteristic(
        service,
        NOTIFY_CHAR_UUID,
        read=True,
        notify=True,
    )

    # Register service
    aioble.register_services(service)
'''
async def writer_loop(write_char, notify_char, conn):
    try:
        while conn.is_connected():
            print("Written by device: ", conn.device)
            writer = await write_char.written()
            data = write_char.read()
            print("Got write:", data)
            if conn.is_connected():
                notify_char.notify(conn, b"ACK: hi")
                print("Notify sent successfully")
                await asyncio.sleep(3)
    except asyncio.CancelledError:
        print("Writer task cancelled")
    except Exception as e:
        print("Error sending notification:", e)

# Main advertising and connection loop
async def connection_handler():
    global write_char, notify_char
    
    while True:
        print("Advertising...")
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

        aioble.log_level=2
        # Register service
        aioble.register_services(service)
        try:
            async with await aioble.advertise(
                ble_advertiseing_interval,
                name="pico2w_ble",
                services=[SERVICE_UUID],
                appearance=ble_apprearance,
                timeout_ms=30000
            ) as conn:
                print("Connected to:", conn.device)
                
                writer_task = asyncio.create_task(writer_loop(write_char, notify_char, conn))

                while conn.is_connected():
                    await asyncio.sleep(0.5)

                print("Connection lost, cancelling writer task")
                writer_task.cancel()
                await asyncio.sleep(1)
            
        except Exception as e:
            print("Advertising or connection failed:", e)

        print("Client disconnected, restarting advertise loop.")
        await asyncio.sleep(1)  # <== give BLE stack some time
        # Fully reset BLE here
        ble = bluetooth.BLE()
        ble.active(False)
        await asyncio.sleep(1)
        ble.active(True)
        
        #run_service()
        
        await asyncio.sleep(1)

# Run the BLE handler
#run_service()
asyncio.run(connection_handler())
