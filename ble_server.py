import uasyncio as asyncio
import aioble
import bluetooth
import os
import struct

aioble.log_level=2
ble_apprearance = 0x0300
ble_advertiseing_interval = 2000

# BLE UUIDs
SERVICE_UUID = bluetooth.UUID(0x1234)     # Environmental Sensing
WRITE_CHAR_UUID = bluetooth.UUID(0x6e40)   # Temperature (write)
NOTIFY_CHAR_UUID = bluetooth.UUID(0x6e41)  # Notify

FILE_DIR = "/"

async def handle_echo(notify_char, conn, data):
    print("received echo request, echo back: ", data)
    notify_char.notify(conn, data)  # Echo back data
    await asyncio.sleep(3)
    
# === File listing formatter ===
def handle_list_files(notify_char, conn):
    entries = []
    for name in os.listdir(FILE_DIR):
        try:
            print("stat:", name)
            size = os.stat(FILE_DIR + "/" + name)[6]
            entries.append(f"{name},{size}")
        except:
            continue
    response = ";".join(entries)
    notify_char.notify(conn, response.encode())
    await asyncio.sleep(3)
    
async def handle_get_file(notify_char, conn, data):
    print("get file: ", data)
    try:
        with open(data, "rb") as f:
            buf = f.read()
            notify_char.notify(conn, buf)  # Echo file content
            await asyncio.sleep(3)
    except OSError as e:
        print("Failed to read file:", e)
      
async def handle_save_file(notify_char, conn, data):
    if len(data) < 21:
        print(f"[UPLOAD] err: too short")
        notify_char.notify(conn, b"ERR:Too short")
        return

    md5name = binascii.hexlify(data[:16]).decode()
    
    file_size = struct.unpack(">I", data[16:20])[0]
    content = data[20:]
    content_size = len(content)
    if content_size != file_size:
        print(f"[UPLOAD] Filename={md5name}: wrong size {file_size} != {content_size}")
        notify_char.notify(conn, b"ERR:Size mismatch")
        return
    
    print(f"[UPLOAD] Filename={md5name} Size={file_size}")

    try:
        with open(FILE_DIR + "/" + md5name, "wb") as f:
            f.write(content)
        notify_char.notify(conn, b"ACK:OK")
    except Exception as e:
        notify_char.notify(conn, "ERR:{str(e)}".encode())
            
async def handle_delete_file(notify_char, conn, data):
    if len(data) != 20:
        print(f"[DELETE] wrong length {data}")
        notify_char.notify(conn, b"ERR:WRONG REQUEST")
        return
    
    try:
        md5hash = binascii.hexlify(data[:16]).decode()
        req = struct.unpack(">I", data[16:])[0]
    
        filepath = FILE_DIR + "/" + filename
        
        if not os.path.exists(filepath):
            print(f"[DELETE] {filepath}: not found")
            notify_char.notify(conn, b"ERR:NOT_FOUND")
            return
        
        size = os.stat(filename)[6]
        
        if req != size:
            print(f"[DELETE] {filepath}: wrong size {req} != {size}")
            notify_char.notify(conn, b"ERR:WRONG REQUEST")
            return
    
        os.remove(filepath)
        notify_char.notify(conn, b"ACK:DELETED")
            
    except Exception as e:
        print("delete file: ", e)
            
async def handle_data(notify_char, conn, data):
    method = data[0]
    print("received method: ", method)

    # Method 0: Echo (health check)
    if method == 0:
        await handle_echo(notify_char, conn, data[1:])

    # Method 1: File upload
    elif method == 1:
        await handle_save_file(notify_char, conn, data[1:])

    # Method 2: File delete
    elif method == 2:
        await handle_delete_file(notify_char, conn, data[1:])

    # Method 3: File list
    elif method == 3:
        handle_list_files(notify_char, conn)
        
    # Method 4: File list
    elif method == 4:
        await handle_get_file(notify_char, conn, data[1:])

    else:
        await send_notify(b"ERR:Unknown method")


async def writer_loop(write_char, notify_char, conn):
    try:
        while conn.is_connected():
            print("Written by device: ", conn.device)
            writer = await write_char.written()
            data = write_char.read()
            print("Got write:", data)
            if conn.is_connected():
                await handle_data(notify_char, conn, data)
            else:
                print("Connection lost")
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
                appearance=ble_apprearance
            ) as conn:
                print("Connected to:", conn.device)
                
                writer_task = asyncio.create_task(writer_loop(write_char, notify_char,conn))

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
